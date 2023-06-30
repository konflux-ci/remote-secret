//
// Copyright (c) 2021 Red Hat, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"
	"github.com/go-logr/logr"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ secretstorage.SecretStorage = (*AwsSecretStorage)(nil)

var (
	errGotNilSecret            = errors.New("got nil secret from aws secretmanager")
	errASWSecretCreationFailed = errors.New("failed to create the secret in AWS storage ")
	errASWSecretDeletionFailed = errors.New("failed to delete the secret from AWS storage ")
	errAWSInvalidRequest       = errors.New("invalid request reported when making request to aws")
	errAWSUnknownError         = errors.New("not able to get secret from the aws storage for some unknown reason")
)

// awsClient is an interface grouping methods from aws secretsmanager.Client that we need for implementation of our aws tokenstorage
// This is not complete list of secretsmanager.Client methods
// This is mostly done for testing purpose so we can easily mock the aws client
type awsClient interface {
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

type AwsSecretStorage struct {
	SpiInstanceId string
	Config        *aws.Config

	secretNameFormat string
	client           awsClient
}

func (s *AwsSecretStorage) Initialize(ctx context.Context) error {
	lg(ctx).Info("initializing AWS token storage")

	s.client = secretsmanager.NewFromConfig(*s.Config)
	s.secretNameFormat = s.initSecretNameFormat()

	if errCheck := s.checkCredentials(ctx); errCheck != nil {
		return fmt.Errorf("failed to initialize AWS tokenstorage: %w", errCheck)
	}

	return nil
}

func (s *AwsSecretStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {
	dbgLog := lg(ctx).V(logs.DebugLevel).WithValues("secretID", id)

	dbgLog.Info("storing data")

	secretName := s.generateAwsSecretName(&id)

	dbgLog = dbgLog.WithValues("secretname", secretName)
	ctx = log.IntoContext(ctx, dbgLog)

	errCreate := s.createOrUpdateAwsSecret(ctx, secretName, data)
	if errCreate != nil {
		dbgLog.Error(errCreate, "secret creation failed")
		return errASWSecretCreationFailed
	}
	return nil
}

func (s *AwsSecretStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {
	dbgLog := lg(ctx).V(logs.DebugLevel).WithValues("secretID", id)

	secretName := s.generateAwsSecretName(&id)
	dbgLog.Info("getting the token", "secretname", secretName, "secretId", id)
	getResult, err := s.getAwsSecret(ctx, secretName)

	if err != nil {
		var awsError smithy.APIError
		if errors.As(err, &awsError) {
			if notFoundErr, ok := awsError.(*types.ResourceNotFoundException); ok {
				secretData, migrationErr := s.tryMigrateSecret(ctx, id) // this migration is just temporary
				if migrationErr != nil {
					dbgLog.Error(migrationErr, "something went wrong during migration")
				}
				if secretData != nil {
					dbgLog.Info("secret successfully migrated", "secretid", id)
					return secretData, nil
				} else {
					dbgLog.Error(notFoundErr, "secret not found in aws storage")
					return nil, fmt.Errorf("%w: %s", secretstorage.NotFoundError, notFoundErr.ErrorMessage())
				}
			}

			if invalidRequestErr, ok := awsError.(*types.InvalidRequestException); ok {
				dbgLog.Error(invalidRequestErr, "invalid request to aws secret storage")
				return nil, fmt.Errorf("%w. message: %s", errAWSInvalidRequest, invalidRequestErr.ErrorMessage())
			}
		}

		dbgLog.Error(err, "unknown error on reading aws secret storage")
		return nil, errAWSUnknownError
	}

	return getResult.SecretBinary, nil
}

func (s *AwsSecretStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {
	dbgLog := lg(ctx).V(logs.DebugLevel).WithValues("secretID", id)
	dbgLog.Info("deleting the token")

	secretName := s.generateAwsSecretName(&id)
	errDelete := s.deleteAwsSecret(ctx, secretName)
	if errDelete != nil {
		dbgLog.Error(errDelete, "secret deletion failed")
		return errASWSecretDeletionFailed
	}
	return nil
}

func (s *AwsSecretStorage) checkCredentials(ctx context.Context) error {
	// let's try to do simple request to verify that credentials are correct or fail fast
	_, err := s.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{MaxResults: aws.Int32(1)})
	if err != nil {
		return fmt.Errorf("failed to list the secrets to check the AWS client is properly configured: %w", err)
	}
	return nil
}

func (s *AwsSecretStorage) createOrUpdateAwsSecret(ctx context.Context, name *string, data []byte) error {
	dbgLog := lg(ctx)
	dbgLog.Info("creating the AWS secret")

	createInput := &secretsmanager.CreateSecretInput{
		Name:         name,
		SecretBinary: data,
	}

	_, errCreate := s.client.CreateSecret(ctx, createInput)
	if errCreate != nil {
		var awsError smithy.APIError
		if errors.As(errCreate, &awsError) {
			// if secret with same name already exists in AWS, we try to update it
			if errAlreadyExists, ok := awsError.(*types.ResourceExistsException); ok {
				dbgLog.Info("AWS secret already exists, trying to update")
				updateErr := s.updateAwsSecret(ctx, createInput.Name, createInput.SecretBinary)
				if updateErr != nil {
					return fmt.Errorf("failed to update the secret: %w", errAlreadyExists)
				}
				return nil
			}
		}
		return fmt.Errorf("error creating the secret: %w", errCreate)
	}

	return nil
}

func (s *AwsSecretStorage) updateAwsSecret(ctx context.Context, name *string, data []byte) error {
	lg(ctx).Info("updating the AWS secret")

	awsSecret, errGet := s.getAwsSecret(ctx, name)
	if errGet != nil {
		return fmt.Errorf("failed to get the secret '%s' to update it in aws secretmanager: %w", *name, errGet)
	}

	updateInput := &secretsmanager.UpdateSecretInput{SecretId: awsSecret.ARN, SecretBinary: data}
	_, errUpdate := s.client.UpdateSecret(ctx, updateInput)
	if errUpdate != nil {
		return fmt.Errorf("failed to update the secret '%s' in aws secretmanager: %w", *name, errUpdate)
	}
	return nil
}

func (s *AwsSecretStorage) getAwsSecret(ctx context.Context, secretName *string) (*secretsmanager.GetSecretValueOutput, error) {
	lg(ctx).V(logs.DebugLevel).Info("getting AWS secret", "secretname", secretName)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: secretName,
	}

	awsSecret, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get the secret '%s' from aws secretmanager: %w", *secretName, err)
	}
	if awsSecret == nil {
		return nil, fmt.Errorf("%w: secretname=%s", errGotNilSecret, *secretName)
	}
	return awsSecret, nil
}

func (s *AwsSecretStorage) deleteAwsSecret(ctx context.Context, secretName *string) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId:                   secretName,
		ForceDeleteWithoutRecovery: aws.Bool(true),
	}

	_, err := s.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("error deleting AWS secret: %w", err)
	}
	return nil
}

func (s *AwsSecretStorage) generateAwsSecretName(secretId *secretstorage.SecretID) *string {
	return aws.String(fmt.Sprintf(s.secretNameFormat, secretId.Uid))
}

func (s *AwsSecretStorage) initSecretNameFormat() string {
	if s.SpiInstanceId == "" {
		return "%s"
	} else {
		return s.SpiInstanceId + "/%s"
	}
}

// tryMigrateSecret tries to migrate secret data from old name (derived from k8s object namespace and name) to new one (derived from k8s object uid).
// returning byte data means the secret was successfully migrated to new location
func (s *AwsSecretStorage) tryMigrateSecret(ctx context.Context, secretId secretstorage.SecretID) ([]byte, error) {
	lg(ctx).Info("trying to migrate AWS secret", "secretid", secretId)
	dbLog := lg(ctx).V(logs.DebugLevel).WithValues("secretId", secretId)

	legacyNameFormat := "%s/%s"
	if s.SpiInstanceId != "" {
		legacyNameFormat = s.SpiInstanceId + "/%s/%s"
	}
	legacySecretName := aws.String(fmt.Sprintf(legacyNameFormat, secretId.Namespace, secretId.Name))

	// first try to get legacy secret, if it is not there, we just stop migration
	getOutput, errGetSecret := s.getAwsSecret(ctx, legacySecretName)
	if errGetSecret != nil {
		var awsError smithy.APIError
		if errors.As(errGetSecret, &awsError) {
			if _, ok := awsError.(*types.ResourceNotFoundException); ok {
				dbLog.Info("no legacy secret found, nothing to do")
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to get the legacy secret during migration: %w", errGetSecret)
	}

	newSecretName := s.generateAwsSecretName(&secretId)
	dbLog.Info("found legacy secret, migrating to new name", "legacy_name", legacySecretName, "new_name", newSecretName)

	// create secret with new name
	errCreate := s.createOrUpdateAwsSecret(ctx, newSecretName, getOutput.SecretBinary)
	if errCreate != nil {
		return nil, fmt.Errorf("failed to create the new secret during migration: %w", errGetSecret)
	}

	errDelete := s.deleteAwsSecret(ctx, legacySecretName)
	if errDelete != nil {
		lg(ctx).Error(errDelete, "failed to delete legacy secret during migration")
	}

	return getOutput.SecretBinary, nil
}

func lg(ctx context.Context) logr.Logger {
	return log.FromContext(ctx, "secretstorage", "AWS")
}
