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

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/es"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/memorystorage"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/awsstorage/awscli"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/vaultstorage/vaultcli"
)

var (
	errUnsupportedSecretStorage = errors.New("unsupported secret storage type")
	errNilSecretStorage         = errors.New("nil secret storage")
)

func CreateInitializedSecretStorage(ctx context.Context, args *CommonCliArgs) (secretstorage.SecretStorage, error) {
	var storage secretstorage.SecretStorage
	var err error

	switch args.TokenStorage {
	case VaultTokenStorage:
		storage, err = vaultcli.CreateVaultStorage(ctx, &args.VaultCliArgs)
	case AWSTokenStorage:
		storage, err = awscli.NewAwsSecretStorage(ctx, args.InstanceId, &args.AWSCliArgs)
	case ESSecretStorage:
		storage, err = es.NewESSecretStorage(ctx, args.ConfigFile)
	case InMemoryStorage:
		storage = &memorystorage.MemoryStorage{}
	default:
		return nil, fmt.Errorf("%w '%s'", errUnsupportedSecretStorage, args.TokenStorage)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create the secret storage '%s': %w", args.TokenStorage, err)
	}

	if storage == nil {
		return nil, fmt.Errorf("%w: '%s'", errNilSecretStorage, args.TokenStorage)
	}

	if err = storage.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize the secret storage '%s': %w", args.TokenStorage, err)
	}

	return storage, nil
}
