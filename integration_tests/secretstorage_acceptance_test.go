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

package integrationtests

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/awsstorage/awscli"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/memorystorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/vaultstorage"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// Testsuite that runs same tests against multiple tokenstorage implementations.
// Tests cares just about TokenStorage interface and makes sure that implementations behaves the same.
// Runs against real storages, see comments on Test* functions for more details.

// TestVaultStorage runs against local in-memory Vault. No external dependency or setup is needed, test runs everytime.
func TestVaultStorage(t *testing.T) {
	cluster, storage := vaultstorage.CreateTestVaultSecretStorage(t)

	ctx := context.TODO()
	assert.NoError(t, storage.Initialize(ctx))
	defer cluster.Cleanup()

	testStorage(t, ctx, storage)
}

// TestInMemoryStorage runs against our testing in-memory implementation of the TokenStorage.
func TestInMemoryStorage(t *testing.T) {
	storage := &memorystorage.MemoryStorage{}

	ctx := context.TODO()
	assert.NoError(t, storage.Initialize(ctx))

	testStorage(t, ctx, storage)
}

// TestAws runs against real AWS secret manager.
// AWS_CONFIG_FILE and AWS_CREDENTIALS_FILE must be set and point to real files with real credentials for testsuite to properly run. Otherwise test is skipped.
func TestAws(t *testing.T) {
	ctx := context.TODO()

	awsConfig, hasAwsConfig := os.LookupEnv("AWS_CONFIG_FILE")
	awsCreds, hasAwsCreds := os.LookupEnv("AWS_CREDENTIALS_FILE")

	if !hasAwsConfig || !hasAwsCreds {
		t.Log("AWS storage tests didn't run!. to test AWS storage, set AWS_CONFIG_FILE and AWS_CREDENTIALS_FILE env vars")
		return
	}

	if _, err := os.Stat(awsConfig); errors.Is(err, os.ErrNotExist) {
		t.Logf("AWS storage tests didn't run!. AWS_CONFIG_FILE is set, but file does not exists '%s'\n", awsConfig)
		return
	}

	if _, err := os.Stat(awsCreds); errors.Is(err, os.ErrNotExist) {
		t.Logf("AWS storage tests didn't run!. AWS_CREDENTIALS_FILE is set, but file does not exists '%s'\n", awsCreds)
		return
	}

	secretStorage, error := awscli.NewAwsSecretStorage(ctx, "spi-test", &awscli.AWSCliArgs{
		ConfigFile:      awsConfig,
		CredentialsFile: awsCreds,
	})
	assert.NoError(t, error)
	assert.NotNil(t, secretStorage)

	err := secretStorage.Initialize(ctx)
	assert.NoError(t, err)

	testStorage(t, ctx, secretStorage)
}

func testStorage(t *testing.T, ctx context.Context, storage secretstorage.SecretStorage) {
	refreshTestData(t)

	// get non-existing
	gettedSecretData, err := storage.Get(ctx, secretId)
	assert.ErrorIs(t, err, secretstorage.NotFoundError)
	assert.Nil(t, gettedSecretData)

	// delete non-existing
	err = storage.Delete(ctx, secretId)
	assert.NoError(t, err)

	// create
	err = storage.Store(ctx, secretId, testSecretData)
	assert.NoError(t, err)

	// write, let's wait a bit
	time.Sleep(1 * time.Second)

	// get
	actualSecretData, err := storage.Get(ctx, secretId)
	assert.NoError(t, err)
	assert.NotNil(t, actualSecretData)
	assert.EqualValues(t, testSecretData, actualSecretData)

	// update
	err = storage.Store(ctx, secretId, updatedSecretData)
	assert.NoError(t, err)

	// write, let's wait a bit
	time.Sleep(1 * time.Second)

	// get
	gettedSecretData, err = storage.Get(ctx, secretId)
	assert.NoError(t, err)
	assert.NotNil(t, gettedSecretData)
	assert.EqualValues(t, updatedSecretData, gettedSecretData)

	// delete
	err = storage.Delete(ctx, secretId)
	assert.NoError(t, err)

	// write, let's wait a bit
	time.Sleep(1 * time.Second)

	// get deleted
	gettedSecretData, err = storage.Get(ctx, secretId)
	assert.ErrorIs(t, err, secretstorage.NotFoundError)
	assert.Nil(t, gettedSecretData)

}

var (
	secretId          secretstorage.SecretID
	testSecretData    []byte
	updatedSecretData []byte
)

func refreshTestData(t *testing.T) {

	random, _, _ := strings.Cut(string(uuid.NewUUID()), "-")
	secretId = secretstorage.SecretID{Uid: uuid.NewUUID(), Name: "secret" + random, Namespace: "ns" + random}
	testData := map[string]any{
		"username":     "testUsername-" + random,
		"accessToken":  "testAccessToken-" + random,
		"tokenType":    "testTokenType-" + random,
		"refreshToken": "testRefreshToken-" + random,
		"expiry":       rand.Uint64() % 1000,
	}
	bytes, err := secretstorage.SerializeJSON(&testData)
	assert.NoError(t, err)
	testSecretData = bytes
	testData["accessToken"] = "testAccessToken-" + random + "-update"
	bytes, err = secretstorage.SerializeJSON(&testData)
	assert.NoError(t, err)
	updatedSecretData = bytes
}
