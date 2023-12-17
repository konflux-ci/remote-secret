package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var secretStoreTimeMetric = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: config.MetricsNamespace,
	Subsystem: config.MetricsSubsystem,
	Name:      "secret_store_operation_time_seconds",
	Help:      "the time it takes to complete operation with secret data in secret storage",
}, []string{"type", "operation"})

var _ secretstorage.SecretStorage = (*MeteredSecretStorage)(nil)

type MeteredSecretStorage struct {
	SecretStorage     secretstorage.SecretStorage
	StorageType       string
	MetricsRegisterer prometheus.Registerer
	storeMetric       prometheus.Observer
	deleteMetric      prometheus.Observer
	getMetric         prometheus.Observer
}

func (m *MeteredSecretStorage) Initialize(ctx context.Context) error {
	m.storeMetric = secretStoreTimeMetric.WithLabelValues(m.StorageType, "store")
	m.deleteMetric = secretStoreTimeMetric.WithLabelValues(m.StorageType, "delete")
	m.getMetric = secretStoreTimeMetric.WithLabelValues(m.StorageType, "get")
	m.MetricsRegisterer.MustRegister(secretStoreTimeMetric)
	return m.SecretStorage.Initialize(ctx)
}

func (m *MeteredSecretStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {
	lg := log.FromContext(ctx)
	timer := prometheus.NewTimer(m.storeMetric)
	defer timer.ObserveDuration()
	lg.Info("Store->>>> to time")
	return m.SecretStorage.Store(ctx, id, data)
}

func (m *MeteredSecretStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {
	timer := prometheus.NewTimer(m.getMetric)
	defer timer.ObserveDuration()
	return m.SecretStorage.Get(ctx, id)
}

func (m *MeteredSecretStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {
	timer := prometheus.NewTimer(m.deleteMetric)
	defer timer.ObserveDuration()
	return m.SecretStorage.Delete(ctx, id)
}
