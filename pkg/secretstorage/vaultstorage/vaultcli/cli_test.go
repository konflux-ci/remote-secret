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

package vaultcli

import (
	"context"
	"testing"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/vaultstorage"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestVaultStorageConfigFromCliArgs(t *testing.T) {
	//given
	ctx := context.TODO()
	vaultArg := &VaultCliArgs{VaultDataPathPrefix: "spi", VaultAuthMethod: vaultstorage.VaultAuthMethodApprole, VaultAppRoleSecretName: "my-secret", VaultAppRoleSecretNamespace: "my-ns"}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vaultArg.VaultAppRoleSecretName,
			Namespace: vaultArg.VaultAppRoleSecretNamespace,
		},
		Data: map[string][]byte{
			"secret_id": []byte("s-01"),
			"role_id":   []byte("r-01"),
		},
	}

	k := fake.NewClientBuilder().
		WithObjects(secret).
		Build()
	//when
	config, err := VaultStorageConfigFromCliArgs(ctx, vaultArg, k)
	//then
	assert.NoError(t, err)
	assert.Equal(t, "spi", config.DataPathPrefix)
	assert.Equal(t, string(secret.Data["secret_id"]), config.SecretId)
	assert.Equal(t, string(secret.Data["role_id"]), config.RoleId)
}

//TODO more tests for corner cases
