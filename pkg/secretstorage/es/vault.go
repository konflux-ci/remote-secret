package es

import vault "github.com/external-secrets/external-secrets/pkg/provider/vault"

type VaultSecretStorage struct {
	Conntector vault.Connector
}
