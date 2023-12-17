package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/memorystorage"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testData = []byte("test_data")

var testSecretID = secretstorage.SecretID{
	Name:      "testRemoteSecret",
	Namespace: "testNamespace",
}

func TestTimeMetric(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	secretStoreTimeMetric.Reset()

	storage := &MeteredSecretStorage{
		SecretStorage:     &memorystorage.MemoryStorage{},
		StorageType:       "memory",
		MetricsRegisterer: registry,
	}

	err := storage.Initialize(context.TODO())
	assert.NoError(t, err)

	errStore := storage.Store(context.TODO(), testSecretID, testData)
	assert.NoError(t, errStore)
	//assert.True(t, cl.createCalled)

	assert.Equal(t, 1, prometheusTest.CollectAndCount(secretStoreTimeMetric))

	//_, errStore = strg.Get(context.TODO(), testSecretID)
	//assert.NoError(t, errStore)
	//assert.True(t, cl.getCalled)
	//assert.Equal(t, 2, prometheusTest.CollectAndCount(awsStoreTimeMetric))
	//
	//errStore = strg.Delete(context.TODO(), testSecretID)
	//assert.NoError(t, errStore)
	//assert.True(t, cl.deleteCalled)
	//assert.Equal(t, 3, prometheusTest.CollectAndCount(awsStoreTimeMetric))

	//assert.NoError(t, RegisterCommonMetrics(registry))
	//
	//secretStoreTimeMetric.WithLabelValues("foo_operation", "some_reason").Inc()
	//secretStoreTimeMetric.WithLabelValues("foo_operation", "some_other_reason").Inc()
	//
	//count, err := prometheusTest.GatherAndCount(registry, "redhat_appstudio_remotesecret_secret_store_operation_time_seconds")
	//assert.Equal(t, 2, count)
	//assert.NoError(t, err)
}
