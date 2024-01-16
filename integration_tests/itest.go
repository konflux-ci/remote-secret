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

package integrationtests

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"

	. "github.com/onsi/gomega"
	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/memorystorage"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var ITest = struct {
	TestEnvironment       *envtest.Environment
	Context               context.Context //nolint: containedctx // we DO want the context shared across all integration tests...
	Cancel                context.CancelFunc
	Client                client.Client
	ClientFactory         TestClientFactory
	Storage               *ITestStorage
	OperatorConfiguration *config.OperatorConfiguration
	Registry              *prometheus.Registry
}{}

type TestClientFactory struct {
	GetClientImpl             func(ctx context.Context, currentNamespace string, targetSpec *v1beta1.RemoteSecretTarget, targetStatus *v1beta1.TargetStatus) (client.Client, error)
	ServiceAccountChangedImpl func(client.ObjectKey)
}

// GetClient implements bindings.ClientFactory
func (tcf *TestClientFactory) GetClient(ctx context.Context, currentNamespace string, targetSpec *v1beta1.RemoteSecretTarget, targetStatus *v1beta1.TargetStatus) (client.Client, error) {
	if tcf.GetClientImpl == nil {
		return nil, nil
	}
	return tcf.GetClientImpl(ctx, currentNamespace, targetSpec, targetStatus)
}

// ServiceAccountChanged implements bindings.ClientFactory
func (tcf *TestClientFactory) ServiceAccountChanged(sa types.NamespacedName) {
	if tcf.ServiceAccountChangedImpl != nil {
		tcf.ServiceAccountChangedImpl(sa)
	}
}

var _ bindings.ClientFactory = (*TestClientFactory)(nil)

func init() {
	logs.InitDevelLoggers()
}

var _ remotesecretstorage.RemoteSecretStorage = (*ITestStorage)(nil)

// ITestStorage implements RemoteSecretStorage and uses MemoryStorage as a backend.
// Provides additional methods to reset backed storage after each test.
type ITestStorage struct {
	remoteSecretStorage remotesecretstorage.RemoteSecretStorage
	memoryStorage       *memorystorage.MemoryStorage
}

func newITestStorage() *ITestStorage {
	return &ITestStorage{}
}

func (i *ITestStorage) PartialUpdate(ctx context.Context, id *api.RemoteSecret, dataUpdates *remotesecretstorage.SecretData, deleteKeys []string) error {
	err := i.remoteSecretStorage.PartialUpdate(ctx, id, dataUpdates, deleteKeys)
	if err != nil {
		return fmt.Errorf("partial update error: %w", err)
	}
	return nil

}

func (i *ITestStorage) Initialize(ctx context.Context) error {
	i.memoryStorage = &memorystorage.MemoryStorage{}
	i.remoteSecretStorage = remotesecretstorage.NewJSONSerializingRemoteSecretStorage(i.memoryStorage)
	return nil
}

func (i *ITestStorage) Store(ctx context.Context, id *api.RemoteSecret, data *remotesecretstorage.SecretData) error {
	err := i.remoteSecretStorage.Store(ctx, id, data)
	if err != nil {
		return fmt.Errorf("store error: %w", err)
	}
	return nil

}

func (i *ITestStorage) Get(ctx context.Context, id *api.RemoteSecret) (*remotesecretstorage.SecretData, error) {
	data, err := i.remoteSecretStorage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get error: %w", err)
	}
	return data, nil
}

func (i *ITestStorage) Delete(ctx context.Context, id *api.RemoteSecret) error {
	err := i.remoteSecretStorage.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("delete error: %w", err)
	}
	return nil

}

// SecretStorage returns backend storage
func (i *ITestStorage) SecretStorage() secretstorage.SecretStorage {
	return i.memoryStorage
}

// Reset backend storage to the initial state
func (i *ITestStorage) Reset() {
	i.memoryStorage.Reset()
}

// Len return the number of records in the backend storage
func (i *ITestStorage) Len() int {
	return i.memoryStorage.Len()
}

type StatusConditionValue struct {
	Condition string
	Name      string
	Namespace string
	Status    string
	Value     int
}

// ExpectStatusConditionMetric ensure provided Gatherer has a necessary metrics in redhat_appstudio_remotesecret_status_condition
func ExpectStatusConditionMetric(g prometheus.Gatherer, expectedMetrics []*StatusConditionValue) {

	var expected strings.Builder

	expected.WriteString(`
			# HELP redhat_appstudio_remotesecret_status_condition The status condition of a specific RemoteSecret
			# TYPE redhat_appstudio_remotesecret_status_condition gauge`)
	for _, condition := range expectedMetrics {
		expected.WriteString(fmt.Sprintf("\n\t\t\tredhat_appstudio_remotesecret_status_condition{condition=\"%s\",name=\"%s\",namespace=\"%s\",status=\"%s\"} %d", condition.Condition, condition.Name, condition.Namespace, condition.Status, condition.Value))
	}
	expected.WriteString(`
	`)

	err := prometheusTest.GatherAndCompare(g, strings.NewReader(expected.String()), "redhat_appstudio_remotesecret_status_condition")
	Expect(err).NotTo(HaveOccurred())
}
