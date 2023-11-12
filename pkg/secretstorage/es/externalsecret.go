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
	"github.com/go-logr/logr"

	"github.com/external-secrets/external-secrets/pkg/provider/aws"
	"github.com/external-secrets/external-secrets/pkg/provider/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/vault"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"

	"sigs.k8s.io/controller-runtime/pkg/log"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

// init ESO Providers
var (
	_ = fake.Provider{}
	_ = vault.Connector{}
	_ = aws.Provider{}
)

type ESStorage struct {
	ProviderConfig *es.SecretStoreProvider
	storage        es.SecretStore
	provider       es.Provider
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

func (p *ESStorage) Initialize(ctx context.Context) error {

	lg := lg(ctx)
	lg.Info("initializing ES storage")
	p.storage = es.SecretStore{
		Spec: es.SecretStoreSpec{
			Provider: p.ProviderConfig,
		},
	}

	var err error
	p.provider, err = es.GetProvider(&p.storage)
	if err != nil {
		return fmt.Errorf("failed getting provider %w", err)
	}
	lg.V(logs.DebugLevel).Info("initialized ExternalSecret storage", ":", p.storage)
	return nil
}

func (p *ESStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {

	// TODO Kubeclient and namespace, do we need it?
	client, err := p.provider.NewClient(ctx, &p.storage, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed creating new client %w", err)
	}

	secret, err := client.GetSecret(ctx, es.ExternalSecretDataRemoteRef{Key: id.String()})
	if err != nil {
		if errors.Is(err, es.NoSecretErr) {
			lg(ctx).Info("No secret found ", "ID", id.String())
			return nil, secretstorage.NotFoundError
		}
		return nil, fmt.Errorf("failed getting the secret %w", err)
	}

	return secret, nil
}

func (p *ESStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {

	// TODO Kubeclient and namespace, do we need it?
	client, err := p.provider.NewClient(ctx, &p.storage, nil, "")
	if err != nil {
		return fmt.Errorf("failed creating new client %w", err)
	}

	err = client.DeleteSecret(ctx, &PushData{RemoteKey: id.String()})
	if err != nil {
		return fmt.Errorf("failed deleting the secret %w", err)
	}

	// TODO need to close?
	//client.Close(ctx)
	return nil
}

func (p *ESStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {

	// TODO Kubeclient and namespace, do we need it?
	client, err := p.provider.NewClient(ctx, &p.storage, nil, "")
	if err != nil {
		return fmt.Errorf("failed creating new client %w", err)
	}

	// id.String() -> namespace/name
	err = client.PushSecret(ctx, data, &PushData{RemoteKey: id.String()})
	if err != nil {
		return fmt.Errorf("failed storing the secret %w", err)
	}

	return nil
}

func lg(ctx context.Context) logr.Logger {
	return log.FromContext(ctx, "secretstorage", "es")
}
