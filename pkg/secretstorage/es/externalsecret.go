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

package es

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"strings"

	"github.com/external-secrets/external-secrets/pkg/provider/aws"
	"github.com/external-secrets/external-secrets/pkg/provider/fake"
	"github.com/external-secrets/external-secrets/pkg/provider/kubernetes"
	"github.com/external-secrets/external-secrets/pkg/provider/vault"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	es "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

var _ secretstorage.SecretStorage = (*ESStorage)(nil)

// init ESO Providers
var (
	_ = fake.Provider{}
	_ = vault.Connector{}
	_ = aws.Provider{}
	_ = kubernetes.Provider{}
)

type ESStorage struct {
	ProviderConfig *es.SecretStoreProvider
	storage        es.SecretStore
	provider       es.Provider
	log            logr.Logger
	kube           client.Client
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

// SafeDelimiter is the character used as a placeholder for '/'.
const SafeDelimiter = "l-"

// SafeDash is the character used as a placeholder for '-'.
const SafeDash = "lt"

// EncodeName encodes a name to be Kubernetes-safe.
func EncodeName(name string) string {
	// Encode '/' first to ensure it doesn't conflict with other encodings
	safeName := strings.ReplaceAll(name, "/", SafeDelimiter)
	// Then, encode '-' to ensure it doesn't conflict with SafeDelimiter
	safeName = strings.ReplaceAll(safeName, "-", SafeDash)
	return safeName
}

// DecodeName decodes a Kubernetes-safe name to its original form.
func DecodeName(safeName string) string {
	// Decode '-' first to restore it to its original form
	decodedName := strings.ReplaceAll(safeName, SafeDash, "-")
	// Then, decode '/' to restore it to its original form
	decodedName = strings.ReplaceAll(decodedName, SafeDelimiter, "/")
	return decodedName
}

func (p *ESStorage) Initialize(ctx context.Context) error {

	p.log = log.FromContext(ctx)
	p.log.Info("Initialize")
	mgr := ctx.Value("mgr").(manager.Manager)

	p.kube = mgr.GetClient()
	if p.ProviderConfig == nil {
		return fmt.Errorf("failed initializing ExternalSecret provider, ProviderConfig is not set")
	}
	if p.ProviderConfig.Kubernetes != nil {
		p.ProviderConfig.Kubernetes.Server.URL = mgr.GetConfig().Host
	}
	p.storage = es.SecretStore{
		Spec: es.SecretStoreSpec{
			Provider: p.ProviderConfig,
		},
	}

	var err error
	p.provider, err = es.GetProvider(&p.storage)
	if err != nil {
		return fmt.Errorf("failed getting provider %s", err)
	}
	p.log.V(logs.DebugLevel).Info("initialized ExternalSecret storage", ":", p.storage)
	return nil
}

func (p *ESStorage) Get(ctx context.Context, id secretstorage.SecretID) ([]byte, error) {

	// TODO Kubeclient and namespace, do we need it?
	client, err := p.provider.NewClient(ctx, &p.storage, p.kube, id.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed creating new client %s", err)
	}

	secret, err := client.GetSecret(ctx, es.ExternalSecretDataRemoteRef{Key: id.Namespace, Property: id.Name})
	if err != nil {
		if err == es.NoSecretErr || errors.IsNotFound(err) {
			p.log.Info("No secret found ", "ID", id.String())
			return nil, secretstorage.NotFoundError
		}
		return nil, fmt.Errorf("failed getting the secret %s", err)
	}

	return secret, nil
}

func (p *ESStorage) Delete(ctx context.Context, id secretstorage.SecretID) error {

	// TODO Kubeclient and namespace, do we need it?
	client, err := p.provider.NewClient(ctx, &p.storage, p.kube, id.Namespace)
	if err != nil {
		return fmt.Errorf("failed creating new client %s", err)
	}

	err = client.DeleteSecret(ctx, &PushData{RemoteKey: id.Namespace, Property: id.Name})
	if err != nil {
		return fmt.Errorf("failed deleting the secret %s", err)
	}

	// TODO need to close?
	//client.Close(ctx)
	return nil
}

func (p *ESStorage) Store(ctx context.Context, id secretstorage.SecretID, data []byte) error {

	// TODO Kubeclient and namespace, do we need it?
	client, err := p.provider.NewClient(ctx, &p.storage, p.kube, id.Namespace)
	if err != nil {
		return fmt.Errorf("failed creating new client %s", err)
	}

	// id.String() -> namespace/name
	err = client.PushSecret(ctx, data, &PushData{RemoteKey: id.Namespace, Property: id.Name})
	if err != nil {
		return fmt.Errorf("failed storing the secret %s", err)
	}

	return nil
}
