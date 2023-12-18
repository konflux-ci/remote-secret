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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/stretchr/testify/assert"
)

var testData = []byte("test_data")

var testSecretID = secretstorage.SecretID{
	Name:      "testRemoteSecret",
	Namespace: "testNamespace",
}

var FailedToDeleteError = errors.New("failed to delete")
var FailedToGetError = errors.New("failed to get")
var FailedToCreateError = errors.New("failed to create")
var FailedToUpdateError = errors.New("failed to update")
var FailError = errors.New("fail")

func TestInitialize(t *testing.T) {
	ctx := context.TODO()
	awsConfig, _ := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithSharedConfigFiles([]string{"nothing"}),
		awsconfig.WithSharedCredentialsFiles([]string{"nothing"}))
	strg := AwsSecretStorage{
		Config: &awsConfig,
	}

	errInit := strg.Initialize(ctx)

	assert.Error(t, errInit)
}

func TestInitSecretNameFormat(t *testing.T) {
	s := AwsSecretStorage{
		InstanceId: "blabol",
	}
	assert.Contains(t, s.initSecretNameFormat(), "blabol")
}

func TestGenerateSecretName(t *testing.T) {
	s := AwsSecretStorage{
		secretNameFormat: "%s/%s",
	}

	namespace := "foo"
	name := "rs-test"
	secretName := s.generateAwsSecretName(&secretstorage.SecretID{Namespace: namespace, Name: name})

	assert.NotNil(t, secretName)
	assert.Equal(t, *secretName, "foo/rs-test")
}

func TestCheckCredentials(t *testing.T) {
	ctx := context.TODO()
	t.Run("ok check", func(t *testing.T) {
		cl := &mockAwsClient{
			listFn: func(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
				return nil, nil
			},
		}
		strg := newStorage(cl)
		assert.NoError(t, strg.checkCredentials(ctx))
	})

	t.Run("failed check", func(t *testing.T) {
		ctx := context.TODO()
		cl := &mockAwsClient{
			listFn: func(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
				return nil, FailError
			},
		}
		strg := newStorage(cl)
		assert.Error(t, strg.checkCredentials(ctx))
		assert.True(t, cl.listCalled)
	})
}

func TestStore(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx := context.TODO()
		cl := &mockAwsClient{
			createFn: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return nil, nil
			},
		}

		strg := newStorage(cl)

		errStore := strg.Store(ctx, testSecretID, testData)
		assert.NoError(t, errStore)
		assert.True(t, cl.createCalled)
		assert.False(t, cl.updateCalled)
	})

	t.Run("fail create", func(t *testing.T) {
		ctx := context.TODO()
		cl := &mockAwsClient{
			createFn: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return nil, FailedToCreateError
			},
		}

		strg := newStorage(cl)

		errStore := strg.Store(ctx, testSecretID, testData)
		assert.Error(t, errStore)
		assert.True(t, cl.createCalled)
		assert.False(t, cl.updateCalled)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			createFn: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return nil, &types.ResourceExistsException{}
			},
			updateFn: func(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
				return nil, nil
			},
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{ARN: aws.String("awssecretid")}, nil
			},
		}

		strg := newStorage(cl)

		errStore := strg.Store(ctx, testSecretID, testData)
		assert.NoError(t, errStore)
		assert.True(t, cl.createCalled)
		assert.True(t, cl.updateCalled)
		assert.True(t, cl.getCalled)
	})

	t.Run("fail", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			createFn: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return nil, &types.ResourceExistsException{}
			},
			updateFn: func(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
				return nil, FailedToUpdateError
			},
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{ARN: aws.String("awssecretid")}, nil
			},
		}

		strg := newStorage(cl)

		errStore := strg.Store(ctx, testSecretID, testData)
		assert.Error(t, errStore)
		assert.True(t, cl.createCalled)
		assert.True(t, cl.updateCalled)
		assert.True(t, cl.getCalled)
	})

	t.Run("fail get to update", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			createFn: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return nil, &types.ResourceExistsException{}
			},
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, FailedToGetError
			},
		}

		strg := newStorage(cl)

		errStore := strg.Store(ctx, testSecretID, testData)
		assert.Error(t, errStore)
		assert.True(t, cl.createCalled)
		assert.False(t, cl.updateCalled)
		assert.True(t, cl.getCalled)
	})
}

func TestGet(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{ARN: aws.String("awssecretid"), SecretBinary: testData}, nil
			},
		}

		strg := newStorage(cl)

		data, err := strg.Get(ctx, testSecretID)
		assert.NoError(t, err)
		assert.True(t, cl.getCalled)
		assert.Equal(t, testData, data)
	})

	t.Run("got nil secret", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, nil
			},
		}

		strg := newStorage(cl)

		data, err := strg.Get(ctx, testSecretID)
		assert.Error(t, err)
		assert.True(t, cl.getCalled)
		assert.Nil(t, data)
	})

	t.Run("fail to get", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, FailedToGetError
			},
		}

		strg := newStorage(cl)

		data, err := strg.Get(ctx, testSecretID)
		assert.Error(t, err)
		assert.Equal(t, "not able to get secret from the aws storage for some unknown reason", err.Error())
		assert.True(t, cl.getCalled)
		assert.Nil(t, data)
	})

	t.Run("data not found", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, &types.ResourceNotFoundException{Message: aws.String("no such key exists")}
			},
		}

		strg := newStorage(cl)

		data, err := strg.Get(ctx, testSecretID)
		assert.Error(t, err)
		assert.Error(t, err, secretstorage.NotFoundError)
		assert.Equal(t, "not found: failed to get the secret 'testNamespace/testRemoteSecret' from aws secretmanager: ResourceNotFoundException: no such key exists", err.Error())
		assert.True(t, cl.getCalled)
		assert.Nil(t, data)
	})

	t.Run("secret is deleting", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, &types.InvalidRequestException{Message: aws.String("token is scheduled for deletion")}
			},
		}

		strg := newStorage(cl)

		data, err := strg.Get(ctx, testSecretID)
		assert.Error(t, err)
		assert.Equal(t, "invalid request to aws secret storage: failed to get the secret 'testNamespace/testRemoteSecret' from aws secretmanager: InvalidRequestException: token is scheduled for deletion", err.Error())
		assert.Error(t, err, secretstorage.NotFoundError)
		assert.True(t, cl.getCalled)
		assert.Nil(t, data)
	})

	t.Run("fail invalid request", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			getFn: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, &types.InvalidRequestException{Message: aws.String("some failure")}
			},
		}

		strg := newStorage(cl)

		data, err := strg.Get(ctx, testSecretID)
		assert.Error(t, err)
		assert.Error(t, err, secretstorage.NotFoundError)
		assert.True(t, cl.getCalled)
		assert.Nil(t, data)
	})
}

func TestDelete(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			deleteFn: func(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
				return nil, nil
			},
		}

		strg := newStorage(cl)

		errDelete := strg.Delete(ctx, testSecretID)
		assert.NoError(t, errDelete)
		assert.True(t, cl.deleteCalled)
	})

	t.Run("fail", func(t *testing.T) {
		ctx := context.TODO()

		cl := &mockAwsClient{
			deleteFn: func(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
				return nil, FailedToDeleteError
			},
		}

		strg := newStorage(cl)

		errDelete := strg.Delete(ctx, testSecretID)
		assert.Error(t, errDelete)
		assert.True(t, cl.deleteCalled)
	})
}

func newStorage(cl *mockAwsClient) AwsSecretStorage {
	return AwsSecretStorage{
		client:           cl,
		secretNameFormat: "%s/%s",
	}
}

type mockAwsClient struct {
	createFn     func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	createCalled bool

	getFn     func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	getCalled bool

	listFn     func(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	listCalled bool

	updateFn     func(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	updateCalled bool

	deleteFn     func(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
	deleteCalled bool
}

func (c *mockAwsClient) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	c.createCalled = true
	return c.createFn(ctx, params, optFns...)
}
func (c *mockAwsClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	c.getCalled = true
	return c.getFn(ctx, params, optFns...)
}
func (c *mockAwsClient) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	c.listCalled = true
	return c.listFn(ctx, params, optFns...)
}
func (c *mockAwsClient) UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	c.updateCalled = true
	return c.updateFn(ctx, params, optFns...)
}
func (c *mockAwsClient) DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	c.deleteCalled = true
	return c.deleteFn(ctx, params, optFns...)
}
