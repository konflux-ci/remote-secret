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
	"strings"

	"github.com/pkg/errors"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/vaultstorage"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type VaultCliArgs struct {
	VaultHost                      string                       `arg:"--vault-host, env" help:"Mandatory Vault host URL."`
	VaultInsecureTLS               bool                         `arg:"--vault-insecure-tls, env" default:"false" help:"Whether it allows 'insecure' TLS connection to Vault, 'true' is allowing untrusted certificate."`
	VaultAuthMethod                vaultstorage.VaultAuthMethod `arg:"--vault-auth-method, env" default:"approle" help:"Authentication method to Vault token storage. Options: 'kubernetes', 'approle'."`
	VaultKubernetesSATokenFilePath string                       `arg:"--vault-k8s-sa-token-filepath, env" help:"Used with Vault kubernetes authentication. Filepath to kubernetes ServiceAccount token. When empty, Vault configuration uses default k8s path. No need to set when running in k8s deployment, useful mostly for local development."`
	VaultAppRoleSecretName         string                       `arg:"--vault-approle-secret-name, env" default:"vault-approle-remote-secret-operator" help:"Secret name in k8s namespace with approle credentials. Used with Vault approle authentication. Secret should contain 'role_id' and 'secret_id' keys."`
	VaultKubernetesRole            string                       `arg:"--vault-k8s-role, env"  help:"Used with Vault kubernetes authentication. Vault authentication role set for k8s ServiceAccount."`
	VaultDataPathPrefix            string                       `arg:"--vault-data-path-prefix, env" default:"spi" help:"Path prefix in Vault token storage under which all SPI data will be stored. No leading or trailing '/' should be used, it will be trimmed."`
}

//// VaultStorageConfigFromCliArgs returns an instance of the VaultStorageConfig with some fields initialized from
//// the corresponding CLI arguments. Notably, the VaultStorageConfig.MetricsRegisterer is NOT configured, because this
//// cannot be done using just the CLI arguments.
//// func VaultStorageConfigFromCliArgs(args *VaultCliArgs) *vaultstorage.VaultStorageConfig {
//	return &vaultstorage.VaultStorageConfig{
//		Host:                        args.VaultHost,
//		AuthType:                    args.VaultAuthMethod,
//		Insecure:                    args.VaultInsecureTLS,
//		Role:                        args.VaultKubernetesRole,
//		ServiceAccountTokenFilePath: args.VaultKubernetesSATokenFilePath,
//		RoleIdFilePath:              args.VaultApproleRoleIdFilePath,
//		SecretIdFilePath:            args.VaultApproleSecretIdFilePath,
//		DataPathPrefix:              strings.Trim(args.VaultDataPathPrefix, "/"),
//	}
//}

func CreateVaultStorage(ctx context.Context, reader client.Reader, args *VaultCliArgs) (secretstorage.SecretStorage, error) {
	vaultConfig := &vaultstorage.VaultStorageConfig{
		Host:                        args.VaultHost,
		AuthType:                    args.VaultAuthMethod,
		Insecure:                    args.VaultInsecureTLS,
		Role:                        args.VaultKubernetesRole,
		ServiceAccountTokenFilePath: args.VaultKubernetesSATokenFilePath,
		DataPathPrefix:              strings.Trim(args.VaultDataPathPrefix, "/"),
	}

	if args.VaultAuthMethod == vaultstorage.VaultAuthMethodApprole {

		secret := &corev1.Secret{}
		key := client.ObjectKey{Namespace: "remotesecret", Name: args.VaultAppRoleSecretName}
		err := reader.Get(ctx, key, secret)
		if err != nil {
			return &vaultstorage.VaultSecretStorage{
				Config: vaultConfig,
			}, errors.Wrap(err, "failed to read secret")
		}
		vaultConfig.SecretId = string(secret.Data["secret_id"])
		vaultConfig.RoleId = string(secret.Data["role_id"])
	}

	// use the same metrics registry as the controller-runtime
	vaultConfig.MetricsRegisterer = metrics.Registry

	return &vaultstorage.VaultSecretStorage{
		Config: vaultConfig,
	}, nil
}
