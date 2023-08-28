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

package remotesecretstorage

import (
	"context"
	"fmt"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SecretData = map[string][]byte

// RemoteSecretStorage is a specialized secret storage for remote secrets.
type RemoteSecretStorage interface {
	secretstorage.TypedSecretStorage[api.RemoteSecret, SecretData]

	// PartialUpdate merges the data already present in the remote secret with the "dataUpdates".
	// New keys will be added, existing keys updated and keys from the "deleteKeys" array will be
	// removed from the data.
	PartialUpdate(ctx context.Context, id *api.RemoteSecret, dataUpdates *SecretData, deleteKeys []string) error
}

// NewJSONSerializingRemoteSecretStorage is a convenience function to construct a RemoteSecretStorage instance
// based on the provided SecretStorage and serializing the data to JSON for persistence.
// The returned object is an instance of DefaultTypedSecretStorage set up to work with RemoteSecret
// objects as data keys and returning the SecretData instances.
// NOTE that the provided secret storage MUST BE initialized before this call.
func NewJSONSerializingRemoteSecretStorage(secretStorage secretstorage.SecretStorage) RemoteSecretStorage {
	return &remoteSecretStorage{
		DefaultTypedSecretStorage: secretstorage.DefaultTypedSecretStorage[api.RemoteSecret, SecretData]{
			DataTypeName:  "remote secret",
			SecretStorage: secretStorage,
			ToID:          secretstorage.ObjectToID[*api.RemoteSecret],
			Serialize:     secretstorage.SerializeJSON[SecretData],
			Deserialize:   secretstorage.DeserializeJSON[SecretData],
		},
	}
}

// remoteSecretStorage is a private impl of the RemoteSecretStorage interface. The only way of getting an instance of it
// is by calling the NewJSONSerializingRemoteSecretStorage function that properly initializes the instance.
type remoteSecretStorage struct {
	secretstorage.DefaultTypedSecretStorage[api.RemoteSecret, SecretData]
}

var _ RemoteSecretStorage = (*remoteSecretStorage)(nil)

func (rss *remoteSecretStorage) PartialUpdate(ctx context.Context, id *api.RemoteSecret, dataUpdates *SecretData, deleteKeys []string) error {
	currentData, err := rss.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get the remote secret %s for partial update: %w", client.ObjectKeyFromObject(id), err)
	}

	if dataUpdates != nil {
		for k, v := range *dataUpdates {
			(*currentData)[k] = v
		}
	}

	for _, k := range deleteKeys {
		delete(*currentData, k)
	}

	if err := rss.Store(ctx, id, currentData); err != nil {
		return fmt.Errorf("failed to perform the partial update: %w", err)
	}
	return nil
}
