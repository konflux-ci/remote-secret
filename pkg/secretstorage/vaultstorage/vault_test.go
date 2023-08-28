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

package vaultstorage

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/stretchr/testify/assert"
)

func TestMetricCollection(t *testing.T) {
	ctx := context.Background()
	cluster, storage, _, _ := CreateTestVaultSecretStorageWithAuthAndMetrics(t, prometheus.NewPedanticRegistry())
	assert.NoError(t, storage.Initialize(ctx))
	defer cluster.Cleanup()

	_, err := storage.Get(ctx, secretstorage.SecretID{})
	assert.ErrorIs(t, err, secretstorage.NotFoundError)

	assert.Greater(t, prometheusTest.CollectAndCount(vaultRequestCountMetric), 0)
	assert.Greater(t, prometheusTest.CollectAndCount(vaultResponseTimeMetric), 0)
}

func TestExtractData(t *testing.T) {
	t.Run("extracts new format", func(t *testing.T) {
		origBytes := []byte("bytes")
		data := map[string]any{
			"data": map[string]any{
				"bytes": base64.StdEncoding.EncodeToString(origBytes),
			},
		}

		bytes, err := extractByteData(data)
		assert.NoError(t, err)
		assert.Equal(t, origBytes, bytes)
	})
	//t.Run("extracts legacy data", func(t *testing.T) {
	//	origToken := api.Token{
	//		Username:     "username",
	//		AccessToken:  "access token",
	//		TokenType:    "token type",
	//		RefreshToken: "refresh token",
	//		Expiry:       42,
	//	}
	//
	//	// this is the same thing as origToken as it is coming out of the Vault API
	//	data := map[string]interface{}{
	//		"data": map[string]interface{}{
	//			"username":      "username",
	//			"access_token":  "access token",
	//			"token_type":    "token type",
	//			"refresh_token": "refresh token",
	//			"expiry":        json.Number("42"),
	//		},
	//	}
	//
	//	bytes, legacy, err := extractData(data)
	//	assert.NoError(t, err)
	//	assert.True(t, legacy)
	//
	//	// we need to do the comparison in this round-about way because
	//	// serializing the map might not preserve the order of the fields.
	//	token := api.Token{}
	//	assert.NoError(t, json.Unmarshal(bytes, &token))
	//	assert.Equal(t, origToken, token)
	//})
}

func TestExtractByteData(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		bytes := []byte("kachny")

		data := map[string]interface{}{
			"data": map[string]interface{}{
				"bytes": base64.StdEncoding.EncodeToString(bytes),
			},
		}

		extracted, err := extractByteData(data)
		assert.NoError(t, err)
		assert.Equal(t, bytes, extracted)
	})

	t.Run("no data field", func(t *testing.T) {
		data := map[string]interface{}{}

		extracted, err := extractByteData(data)
		assert.Error(t, err)
		assert.ErrorIs(t, err, UnexpectedDataError)
		assert.Nil(t, extracted)
	})

	t.Run("no bytes field", func(t *testing.T) {
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"kachny": "kachny",
			},
		}

		extracted, err := extractByteData(data)
		assert.Error(t, err)
		assert.ErrorIs(t, err, UnexpectedDataError)
		assert.Nil(t, extracted)
	})

	t.Run("no base64 in bytes", func(t *testing.T) {
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"bytes": "kachny",
			},
		}

		extracted, err := extractByteData(data)
		assert.Error(t, err)
		assert.Nil(t, extracted)
	})
}
