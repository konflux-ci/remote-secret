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
	"errors"
	"fmt"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws"
	"github.com/external-secrets/external-secrets/pkg/provider/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault"
	"github.com/go-logr/logr"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

var _ secretstorage.SecretStorage = (*ExternalSecretStorage)(nil)

// init ESO Providers
var (
	_ = fake.Provider{}
	_ = vault.Connector{}
	_ = aws.Provider{}
)

type ExternalSecretStorage struct {
	ProviderConfig   *es.SecretStoreProvider
	InstanceId       string
	Client           client.Client
	storage          es.ClusterSecretStore
	provider         es.Provider
	secretNameFormat string
}

type PushData struct {
	RemoteKey string
	Property  string
}

func (p *PushData) GetRemoteKey() string {
	return p.RemoteKey
}

func (p *PushData) GetProperty() string {
	return p.Property
}

func (p *ExternalSecretStorage) Initialize(ctx context.Context) error {
	lg := lg(ctx)
	lg.Info("initializing ES storage")

	p.storage = es.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{
			Kind:       esv1beta1.ClusterSecretStoreKind,
			APIVersion: esv1beta1.ClusterSecretStoreKindAPIVersion,
		},
		Spec: es.SecretStoreSpec{
			Provider: p.ProviderConfig,
		},
	}

	var err error
	p.provider, err = es.GetProvider(&p.storage)
	if err != nil {
		return fmt.Errorf("failed getting provider %w", err)
	}
	p.secretNameFormat = p.initSecretNameFormat()
	lg.V(logs.DebugLevel).Info("initialized ExternalSecret storage", ":", p.storage)
	return nil
}

func (p *ExternalSecretStorage) Examine(ctx context.Context) error {
	cl, err := p.provider.NewClient(ctx, &p.storage, p.Client, "default")
	lg(ctx).V(logs.DebugLevel).Info("examining ExternalSecret storage")
	if err != nil {
		return fmt.Errorf("failed creating new client %w", err)
	}
	defer func() {
		err = cl.Close(ctx)
		if err != nil {
			lg(ctx).Error(err, "failed closing client")
		}
	}()
	if res, err := cl.Validate(); err != nil || res != 0 {
		return fmt.Errorf("error examining ES tokenstorage %w, validation result is: %s", err, res.String())
	}
	return nil
}

func (p *ExternalSecretStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {
	client, err := p.provider.NewClient(ctx, &p.storage, p.Client, id.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed creating new client %w", err)
	}

	defer func() {
		err = client.Close(ctx)
		if err != nil {
			lg(ctx).Error(err, "failed closing client")
		}
	}()

	secret, err := client.GetSecret(ctx, es.ExternalSecretDataRemoteRef{Key: fmt.Sprintf(p.secretNameFormat, id.Namespace, id.Name)})
	if err != nil {
		if errors.Is(err, es.NoSecretErr) {
			lg(ctx).Info("No secret found ", "ID", id.String())
			return nil, secretstorage.NotFoundError
		}
		return nil, fmt.Errorf("failed getting the secret %w", err)
	}

	return secret, nil
}

func (p *ExternalSecretStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {
	client, err := p.provider.NewClient(ctx, &p.storage, p.Client, id.Namespace)

	if err != nil {
		return fmt.Errorf("failed creating new client %w", err)
	}
	defer func() {
		err = client.Close(ctx)
		if err != nil {
			lg(ctx).Error(err, "failed closing client")
		}
	}()

	err = client.DeleteSecret(ctx, &PushData{RemoteKey: fmt.Sprintf(p.secretNameFormat, id.Namespace, id.Name)})
	if err != nil {
		return fmt.Errorf("failed deleting the secret %w", err)
	}

	return nil
}

func (p *ExternalSecretStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {
	client, err := p.provider.NewClient(ctx, &p.storage, p.Client, id.Namespace)
	if err != nil {
		return fmt.Errorf("failed creating new client %w", err)
	}
	defer func() {
		err = client.Close(ctx)
		if err != nil {
			lg(ctx).Error(err, "failed closing client")
		}
	}()

	// id.String() -> namespace/name
	err = client.PushSecret(ctx, data, &PushData{RemoteKey: fmt.Sprintf(p.secretNameFormat, id.Namespace, id.Name)})
	if err != nil {
		return fmt.Errorf("failed storing the secret %w", err)
	}

	return nil
}

func (p *ExternalSecretStorage) initSecretNameFormat() string {
	if p.InstanceId == "" {
		return "%s/%s"
	} else {
		return p.InstanceId + "/%s/%s"
	}
}

func lg(ctx context.Context) logr.Logger {
	return log.FromContext(ctx, "secretstorage", "es")
}
