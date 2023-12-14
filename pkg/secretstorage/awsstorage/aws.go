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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/cenkalti/backoff/v4"
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
	errAWSUnknownError         = errors.New("not able to get secret from the aws storage for some unknown reason")
	awsStoreTimeMetric         = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: config.MetricsNamespace,
		Subsystem: config.MetricsSubsystem,
		Name:      "aws_store_time_seconds",
		Help:      "the time it takes to store secret data in AWS",
	})
)

const (
	// Reading or creating AWS secret right after the secret with the same ID was deleted may take some time, until the
	// old one is clear completely.
	// Repeats have exponential time between tries, see https://github.com/cenkalti/backoff/blob/v4/exponential.go
	secretCreationRetryCount = 10
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
	InstanceId        string
	Config            *aws.Config
	MetricsRegisterer prometheus.Registerer

	secretNameFormat string
	client           awsClient
}

func (s *AwsSecretStorage) Initialize(ctx context.Context) error {
	lg(ctx).Info("initializing AWS token storage")

	s.client = secretsmanager.NewFromConfig(*s.Config)
	s.secretNameFormat = s.initSecretNameFormat()
	if err := s.initMetrics(ctx); err != nil {
		return fmt.Errorf("failed to initialize AWS metrics: %w", err)
	}

	if errCheck := s.checkCredentials(ctx); errCheck != nil {
		return fmt.Errorf("failed to initialize AWS tokenstorage: %w", errCheck)
	}

	return nil
}

func (s *AwsSecretStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {
	lg := lg(ctx).WithValues("secretId", id)
	lg.V(logs.DebugLevel).Info("storing data")

	ctx = log.IntoContext(ctx, lg)

	start := time.Now()
	errCreate := s.createOrUpdateAwsSecret(ctx, &id, data)
	awsStoreTimeMetric.Observe(time.Since(start).Seconds())

	if errCreate != nil {
		lg.Error(errCreate, "secret creation failed")
		return errASWSecretCreationFailed
	}
	return nil
}

func (s *AwsSecretStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {
	lg := lg(ctx).WithValues("secretId", id)

	secretName := s.generateAwsSecretName(&id)
	lg.V(logs.DebugLevel).Info("getting the token", "secretname", secretName)
	getResult, err := s.getAwsSecret(ctx, secretName)

	if err != nil {
		if isAwsNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", secretstorage.NotFoundError, err.Error())
		} else if isAwsSecretMarkedForDeletionError(err) {
			// data is still there, but secret is marked for deletion. we can return not found error
			lg.Info("secret marked for deletion in aws storage, retuning NotFound error")
			return nil, fmt.Errorf("%w: %s", secretstorage.NotFoundError, "secret is marked for deletion in aws storage")
		} else if isAwsInvalidRequestError(err) {
			lg.Error(err, "invalid request to aws secret storage")
			return nil, fmt.Errorf("invalid request to aws secret storage: %w", err)
		}

		lg.Error(err, "unknown error on reading aws secret storage")
		return nil, errAWSUnknownError
	}
	return getResult.SecretBinary, nil
}

func (s *AwsSecretStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {
	lg := lg(ctx).WithValues("secretId", id)
	lg.V(logs.DebugLevel).Info("deleting the token")

	secretName := s.generateAwsSecretName(&id)
	errDelete := s.deleteAwsSecret(ctx, secretName)
	if errDelete != nil {
		lg.Error(errDelete, "secret deletion failed")
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

func (s *AwsSecretStorage) createOrUpdateAwsSecret(ctx context.Context, secretId *secretstorage.SecretID, data []byte) error {
	lg := lg(ctx)
	lg.V(logs.DebugLevel).Info("creating the AWS secret")

	name := s.generateAwsSecretName(secretId)
	createInput := &secretsmanager.CreateSecretInput{
		Name:         name,
		SecretBinary: data,
		Tags: []types.Tag{
			{
				Key:   aws.String("namespace"),
				Value: aws.String(secretId.Namespace),
			}, {
				Key:   aws.String("name"),
				Value: aws.String(secretId.Name),
			},
		},
	}
	_, errCreate := s.client.CreateSecret(ctx, createInput)
	if errCreate != nil {
		if isAwsResourceExistsError(errCreate) {
			lg.V(logs.DebugLevel).Info("AWS secret already exists, trying to update")
			updateErr := s.updateAwsSecret(ctx, createInput.Name, createInput.SecretBinary)
			if updateErr != nil {
				return fmt.Errorf("failed to update the secret: %w", errCreate)
			}
			return nil
		} else if isAwsScheduledForDeletionError(errCreate) {
			// data with the same key is still there, but it is marked for deletion. let's try to wait for it to be deleted
			if err := s.doCreateWithRetry(ctx, createInput); err != nil {
				return fmt.Errorf("secret creation failed: %w", err)
			}
			return nil
		} else if isAwsInvalidRequestError(errCreate) {
			return fmt.Errorf("invalid creation request to aws secret storage: %w", errCreate)
		}
		return fmt.Errorf("error creating the secret: %w", errCreate)
	}

	return nil
}

func (s *AwsSecretStorage) doCreateWithRetry(ctx context.Context, createInput *secretsmanager.CreateSecretInput) error {
	lg := lg(ctx).WithValues("secretname", createInput.Name)
	err := backoff.Retry(func() error {
		_, errCreate := s.client.CreateSecret(ctx, createInput)
		if errCreate == nil {
			return nil
		}
		if isAwsScheduledForDeletionError(errCreate) {
			lg.Info("AWS secrets conflict found, secrete scheduled for deletion, trying one more time")
			return errCreate //nolint:wrapcheck // no wrapcheck here, we want to retry
		} else if isAwsInvalidRequestError(errCreate) {
			// a different invalid request type error, return as-is and break the retry loop
			lg.Error(errCreate, "invalid creation request to aws secret storage")
			return backoff.Permanent(fmt.Errorf("invalid creation request %w. ", errCreate)) //nolint:wrapcheck // This is an "indication error" to the Backoff framework that is not exposed further.

		}

		// return as-is and break the retry loop
		return backoff.Permanent(fmt.Errorf("error creating the secret: %w", errCreate)) //nolint:wrapcheck // This is an "indication error" to the Backoff framework that is not exposed further.

	}, backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), secretCreationRetryCount), ctx))

	if err != nil {
		return fmt.Errorf("failed to create the secret after %d retries: %w", secretCreationRetryCount, err)
	}
	return nil
}

func (s *AwsSecretStorage) updateAwsSecret(ctx context.Context, name *string, data []byte) error {
	lg(ctx).V(logs.DebugLevel).Info("updating the AWS secret")

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
	return aws.String(fmt.Sprintf(s.secretNameFormat, secretId.Namespace, secretId.Name))
}

func (s *AwsSecretStorage) initSecretNameFormat() string {
	if s.InstanceId == "" {
		return "%s/%s"
	} else {
		return s.InstanceId + "/%s/%s"
	}
}

func (s *AwsSecretStorage) initMetrics(ctx context.Context) error {
	if s.MetricsRegisterer == nil {
		log.FromContext(ctx).Info("no metrics registry configured - metrics collection for AWS is disabled")
		return nil
	}
	if err := s.MetricsRegisterer.Register(awsStoreTimeMetric); err != nil {
		if !errors.As(err, &prometheus.AlreadyRegisteredError{}) {
			return fmt.Errorf("failed to register AWS store time metric: %w", err)
		}
	}
	return nil
}

func lg(ctx context.Context) logr.Logger {
	return log.FromContext(ctx, "secretstorage", "AWS")
}
