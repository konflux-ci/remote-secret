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
	ctrl "sigs.k8s.io/controller-runtime"
	wh "sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

func SetupAllWebhooks(mgr ctrl.Manager, secretStorage secretstorage.SecretStorage) error {
	remoteSecretStorage := remotesecretstorage.NewJSONSerializingRemoteSecretStorage(secretStorage)
	w := &wh.Webhook{
		Handler: &RemoteSecretWebhook{
			Mutator: &RemoteSecretMutator{
				Client:  mgr.GetClient(),
				Storage: remoteSecretStorage,
			},
			Validator: &RemoteSecretValidator{},
			Decoder:   wh.NewDecoder(mgr.GetScheme()),
		},
		RecoverPanic: false,
	}
	mgr.GetWebhookServer().Register("/mutate-appstudio-redhat-com-v1beta1-remotesecret", w)
	return nil
}
