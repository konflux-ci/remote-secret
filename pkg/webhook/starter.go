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

package webhook

import (
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func SetupAllWebhooks(mgr controllerruntime.Manager, secretStorage secretstorage.SecretStorage) error {
	rs := &api.RemoteSecret{}
	return controllerruntime.NewWebhookManagedBy(mgr).
		WithDefaulter(&RemoteSecretMutator{Storage: secretStorage}).
		WithValidator(&RemoteSecretValidator{}).
		For(rs).
		Complete()
}
