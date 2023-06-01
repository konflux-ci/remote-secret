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
	"github.com/redhat-appstudio/remote-secret/pkg/kubernetesclient"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// This is a wrapper around the provided SecretStorage that creates the SPIAccessTokenDataUpdate
// objects on data modifications.
// The supplied secret storage must be initialized explicitly before it can be used by this storage.
type NotifyingRemoteSecretStorage struct {
	ClientFactory kubernetesclient.K8sClientFactory
	SecretStorage secretstorage.SecretStorage
}

var _ secretstorage.SecretStorage = (*NotifyingRemoteSecretStorage)(nil)

// Delete implements SecretStorage
func (s *NotifyingRemoteSecretStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {
	if err := s.SecretStorage.Delete(ctx, id); err != nil {
		return fmt.Errorf("wrapped storage error: %w", err)
	}

	return s.createDataUpdate(ctx, id)
}

// Get implements SecretStorage
func (s *NotifyingRemoteSecretStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {
	var data []byte
	var err error

	if data, err = s.SecretStorage.Get(ctx, id); err != nil {
		return []byte{}, fmt.Errorf("wrapped storage error: %w", err)
	}
	return data, nil
}

// Initialize implements SecretStorage. It is a noop.
func (s *NotifyingRemoteSecretStorage) Initialize(ctx context.Context) error {
	return nil
}

// Store implements SecretStorage
func (s *NotifyingRemoteSecretStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {
	if err := s.SecretStorage.Store(ctx, id, data); err != nil {
		return fmt.Errorf("wrapped storage error: %w", err)
	}

	return s.createDataUpdate(ctx, id)
}

func (s *NotifyingRemoteSecretStorage) createDataUpdate(ctx context.Context, id secretstorage.SecretID) error {

	lg := log.FromContext(ctx)
	lg.Info("Adding label to RemoteSecret")
	cl, err := s.ClientFactory.CreateClient(ctx)

	if err != nil {
		return fmt.Errorf("failed to create the k8s client to use: %w", err)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		remoteSecret := &api.RemoteSecret{}
		if err := cl.Get(ctx, client.ObjectKey{Name: id.Name, Namespace: id.Namespace}, remoteSecret); err != nil {
			return fmt.Errorf("failed to get the RemoteSecret object %s/%s: %w", id.Namespace, id.Name, err)
		}
		if remoteSecret.Labels == nil {
			remoteSecret.Labels = make(map[string]string)
		}

		remoteSecret.Labels["secretuploaded"] = "true"
		return cl.Update(context.TODO(), remoteSecret)
	})
	if retryErr != nil {
		return fmt.Errorf("update failed: %v", retryErr)
	}
	lg.Info("Adding label to RemoteSecret - ok")
	return nil
}
