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

package es

import (
	"context"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

func NewESSecretStorage(_ context.Context, client client.Client, instanceId string, providerConfJSON string) (secretstorage.SecretStorage, error) {
	providerConf := &es.SecretStoreProvider{}
	err := json.Unmarshal([]byte(providerConfJSON), providerConf)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling string: %w", err)
	}

	return &ExternalSecretStorage{ProviderConfig: providerConf, InstanceId: instanceId, Client: client}, nil
}
