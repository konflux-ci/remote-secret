//
// Copyright (c) 2023 Red Hat, Inc.
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

package metrics

import (
	"context"
	"testing"

	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/stretchr/testify/assert"
)

var testData = []byte("test_data")
var testSecretID = secretstorage.SecretID{
	Name:      "testRemoteSecret",
	Namespace: "testNamespace",
}

func TestCallParentStorage(t *testing.T) {
	t.Run("Initialize method", func(t *testing.T) {
		registry := prometheus.NewPedanticRegistry()
		dummyStorage := NewDummySecretStorage()
		strg := &MeteredSecretStorage{
			SecretStorage:     dummyStorage,
			StorageType:       "dummy",
			MetricsRegisterer: registry,
		}
		err := strg.Initialize(context.TODO())
		assert.NoError(t, err)
		assert.True(t, dummyStorage.InitializeCalled)
		assert.False(t, dummyStorage.StoreCalled)
		assert.False(t, dummyStorage.DeleteCalled)
		assert.False(t, dummyStorage.GetCalled)
	})

	t.Run("Store method", func(t *testing.T) {
		registry := prometheus.NewPedanticRegistry()
		dummyStorage := NewDummySecretStorage()
		strg := &MeteredSecretStorage{
			SecretStorage:     dummyStorage,
			StorageType:       "dummy",
			MetricsRegisterer: registry,
		}
		err := strg.Initialize(context.TODO())
		assert.NoError(t, err)
		err = strg.Store(context.TODO(), testSecretID, testData)
		assert.NoError(t, err)

		assert.True(t, dummyStorage.InitializeCalled)
		assert.True(t, dummyStorage.StoreCalled)
		assert.False(t, dummyStorage.DeleteCalled)
		assert.False(t, dummyStorage.GetCalled)
	})

	t.Run("Delete method", func(t *testing.T) {
		registry := prometheus.NewPedanticRegistry()
		dummyStorage := NewDummySecretStorage()
		strg := &MeteredSecretStorage{
			SecretStorage:     dummyStorage,
			StorageType:       "dummy",
			MetricsRegisterer: registry,
		}
		err := strg.Initialize(context.TODO())
		assert.NoError(t, err)
		err = strg.Delete(context.TODO(), testSecretID)
		assert.NoError(t, err)

		assert.True(t, dummyStorage.InitializeCalled)
		assert.False(t, dummyStorage.StoreCalled)
		assert.True(t, dummyStorage.DeleteCalled)
		assert.False(t, dummyStorage.GetCalled)
	})

	t.Run("Get method", func(t *testing.T) {
		registry := prometheus.NewPedanticRegistry()
		dummyStorage := NewDummySecretStorage()
		strg := &MeteredSecretStorage{
			SecretStorage:     dummyStorage,
			StorageType:       "dummy",
			MetricsRegisterer: registry,
		}
		err := strg.Initialize(context.TODO())
		assert.NoError(t, err)
		_, err = strg.Get(context.TODO(), testSecretID)
		assert.NoError(t, err)

		assert.True(t, dummyStorage.InitializeCalled)
		assert.False(t, dummyStorage.StoreCalled)
		assert.False(t, dummyStorage.DeleteCalled)
		assert.True(t, dummyStorage.GetCalled)
	})
}

func TestMetricsCollection(t *testing.T) {

	tests := []struct {
		name                 string
		SecretStorage        secretstorage.SecretStorage
		StorageType          string
		Registry             *prometheus.Registry
		action               func(ctx context.Context, storage *MeteredSecretStorage) error
		wantLabels           map[string]string
		expectedRequestValue int
	}{
		{
			name:          "store_operation",
			Registry:      prometheus.NewPedanticRegistry(),
			SecretStorage: NewDummySecretStorage(),
			StorageType:   "dummy",
			action: func(ctx context.Context, storage *MeteredSecretStorage) error {
				return storage.Store(ctx, testSecretID, testData)
			},
			wantLabels:           map[string]string{"type": "dummy", "operation": "store"},
			expectedRequestValue: 1,
		},
		{
			name:          "get_operation",
			Registry:      prometheus.NewPedanticRegistry(),
			SecretStorage: NewDummySecretStorage(),
			StorageType:   "dummy",
			action: func(ctx context.Context, storage *MeteredSecretStorage) error {
				_, err := storage.Get(ctx, testSecretID)
				return err
			},
			wantLabels:           map[string]string{"type": "dummy", "operation": "get"},
			expectedRequestValue: 1,
		},
		{
			name:          "delete_operation",
			Registry:      prometheus.NewPedanticRegistry(),
			SecretStorage: NewDummySecretStorage(),
			StorageType:   "dummy",
			action: func(ctx context.Context, storage *MeteredSecretStorage) error {
				return storage.Delete(ctx, testSecretID)
			},
			wantLabels:           map[string]string{"type": "dummy", "operation": "delete"},
			expectedRequestValue: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MeteredSecretStorage{
				SecretStorage:     tt.SecretStorage,
				MetricsRegisterer: tt.Registry,
				StorageType:       tt.StorageType,
			}
			SecretStoreTimeMetric.Reset()
			ctx := context.Background()
			err := storage.Initialize(ctx)
			assert.NoError(t, err)
			err = tt.action(ctx, storage)
			assert.NoError(t, err)

			AssertHistogramTotalCount(t, tt.Registry, "redhat_appstudio_remotesecret_secret_store_operation_time_seconds", tt.wantLabels, tt.expectedRequestValue)
		})
	}

}

// DummySecretStorage is a mock implementation of SecretStorage
type DummySecretStorage struct {
	InitializeCalled bool
	StoreCalled      bool
	GetCalled        bool
	DeleteCalled     bool
}

// NewMockSecretStorage creates a new instance of DummySecretStorage
func NewDummySecretStorage() *DummySecretStorage {
	return &DummySecretStorage{}
}

// Initialize is a mocked implementation of the Initialize method
func (m *DummySecretStorage) Initialize(ctx context.Context) error {
	m.InitializeCalled = true
	return nil
}

// Store is a mocked implementation of the Store method
func (m *DummySecretStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {
	m.StoreCalled = true
	return nil
}

func (m *DummySecretStorage) Examine(ctx context.Context) error {
	return nil
}

// Get is a mocked implementation of the Get method
func (m *DummySecretStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {
	m.GetCalled = true
	return nil, nil
}

// Delete is a mocked implementation of the Delete method
func (m *DummySecretStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {
	m.DeleteCalled = true
	return nil
}

func AssertHistogramTotalCount(t *testing.T, g prometheus.Gatherer, name string, labelFilter map[string]string, wantCount int) {
	metrics, err := g.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %s", err)
	}
	counterSum := 0
	for _, mf := range metrics {
		if mf.GetName() != name {
			continue // Ignore other metrics.
		}
		for _, metric := range mf.GetMetric() {
			if !LabelsMatch(metric, labelFilter) {
				continue
			}
			counterSum += int(metric.GetHistogram().GetSampleCount())
		}
	}
	if wantCount != counterSum {
		t.Errorf("Wanted count %d, got %d for metric %s with labels %#+v", wantCount, counterSum, name, labelFilter)
		for _, mf := range metrics {
			if mf.GetName() == name {
				for _, metric := range mf.GetMetric() {
					t.Logf("\tnear match: %s\n", metric.String())
				}
			}
		}
	}
}

func LabelsMatch(metric *dto.Metric, labelFilter map[string]string) bool {
	metricLabels := map[string]string{}

	for _, labelPair := range metric.Label {
		metricLabels[labelPair.GetName()] = labelPair.GetValue()
	}

	// length comparison then match key to values in the maps
	if len(labelFilter) > len(metricLabels) {
		return false
	}

	for labelName, labelValue := range labelFilter {
		if value, ok := metricLabels[labelName]; !ok || value != labelValue {
			return false
		}
	}

	return true
}
