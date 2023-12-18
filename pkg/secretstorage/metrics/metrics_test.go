package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/stretchr/testify/assert"
	"testing"
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
