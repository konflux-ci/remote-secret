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
	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
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
	Storage               remotesecretstorage.RemoteSecretStorage
	OperatorConfiguration *config.OperatorConfiguration
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
