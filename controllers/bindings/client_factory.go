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

package bindings

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	auth "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RemoteSecretAuthServiceAccountLabel = "appstudio.redhat.com/remotesecret-auth-sa"
	DefaultMaxClientCacheTTL            = 30 * time.Second
)

var (
	noAuthServiceAccountFound                         = errors.New("No service account labeled with '" + RemoteSecretAuthServiceAccountLabel + "' was found that could be used to authenticate deployments to targets in the local cluster")
	onlyOneAuthServiceAccountPerNamespaceAllowed      = errors.New("there can be only one service account labeled with '" + RemoteSecretAuthServiceAccountLabel + "' in a namespace")
	noKubeConfigSpecifiedForConnectionToRemoteCluster = errors.New("a secret with kubeconfig with credentials for connecting to a remote cluster is required")
)

// ClientFactory is a helper interface for the RemoteSecretReconciler that creates clients that are able to deploy to remote secret targets. The default (and only)
// implementation is the CachingClientFactory but is hidden behind an interface so that this can be mocked out in the tests.
type ClientFactory interface {
	// GetClient returns a client that can be used to deploy to a target described by the targetSpec and targetStatus from a remote secret in the provided namespace
	GetClient(ctx context.Context, currentNamespace string, targetSpec *v1beta1.RemoteSecretTarget, targetStatus *v1beta1.TargetStatus) (client.Client, error)
	// ServiceAccountChanged signals to the client factory that give service account changed. The client factory might react by revoking the client associated with
	// the service account from a cache, if any, etc.
	ServiceAccountChanged(sa client.ObjectKey)
}

// LocalClusterConnectionDetails provides the client and configuration for connecting to the local cluster
type LocalClusterConnectionDetails struct {
	Client client.Client
	Config *rest.Config
}

type CachingClientFactory struct {
	// LocalCluster provides the client and configuration for connecting to the local cluster
	LocalCluster LocalClusterConnectionDetails
	// ClientConfigurationInitializer is given the opportunity to configure the rest configuration and client options before
	// the client is created in the factory.
	ClientConfigurationInitializer func(cfg *rest.Config, opts *client.Options)

	// MaxClientCacheTTL is the duration after which the cached clients time out and need to be re-initialized. This is
	// necessary for optimizing the memory consumption versus the performance of the clients.
	MaxClientCacheTTL time.Duration

	cache *cache.Expiring
}

var _ ClientFactory = (*CachingClientFactory)(nil)

type cacheKey struct {
	kubeConfigHash    string
	serviceAccountKey client.ObjectKey
}

func (cf *CachingClientFactory) ServiceAccountChanged(sa client.ObjectKey) {
	if cf.cache == nil {
		return
	}

	key := cacheKey{
		serviceAccountKey: sa,
	}

	cf.cache.Delete(key)
}

func (cf *CachingClientFactory) GetClient(ctx context.Context, currentNamespace string, targetSpec *v1beta1.RemoteSecretTarget, targetStatus *v1beta1.TargetStatus) (client.Client, error) {
	var apiUrl string
	var kubeConfigSecretName string
	var targetNamespace string
	if targetSpec != nil {
		apiUrl = targetSpec.ApiUrl
		kubeConfigSecretName = targetSpec.ClusterCredentialsSecret
		targetNamespace = targetSpec.Namespace
	} else if targetStatus != nil {
		apiUrl = targetStatus.ApiUrl
		kubeConfigSecretName = targetStatus.ClusterCredentialsSecret
		targetNamespace = targetStatus.Namespace
	}

	localConnection := false
	if apiUrl == "" {
		apiUrl = cf.LocalCluster.Config.Host
		localConnection = true
	} else {
		localConnection = apiUrl == cf.LocalCluster.Config.Host
	}

	if localConnection && currentNamespace == targetNamespace {
		// the target namespace is the namespace where the remote secret lives. This is always allowed.
		return cf.LocalCluster.Client, nil
	}

	if cf.cache == nil {
		cf.cache = cache.NewExpiring()
	}

	var err error
	var configGetter restConfigGetter
	// if a kubeconfig was specified, use it regardless of whether we're connecting to a remote cluster or the local cluster.
	if kubeConfigSecretName != "" {
		configGetter = &kubeConfigRestConfigGetter{
			ApiUrl:               apiUrl,
			Client:               cf.LocalCluster.Client,
			CurrentNamespace:     currentNamespace,
			KubeConfigSecretName: kubeConfigSecretName,
		}
	} else if localConnection {
		configGetter = &inNamespaceServiceAccountRestConfigGetter{
			CurrentNamespace: currentNamespace,
			Client:           cf.LocalCluster.Client,
			Config:           cf.LocalCluster.Config,
		}
	} else {
		return nil, noKubeConfigSpecifiedForConnectionToRemoteCluster
	}

	key, err := configGetter.GetCacheKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to determine whether the client is already cached: %w", err)
	}
	cl, ok := cf.cache.Get(key)
	if !ok {
		cfg, ttl, err := configGetter.GetRestConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to construct REST configuration for the kubernetes client to deploy to a target: %w", err)
		}

		scheme := runtime.NewScheme()
		if err := corev1.AddToScheme(scheme); err != nil {
			return nil, fmt.Errorf("failed to initialize a new scheme with core objects. weird: %w", err)
		}

		opts := client.Options{
			Scheme: scheme,
		}

		if cf.ClientConfigurationInitializer != nil {
			cf.ClientConfigurationInitializer(cfg, &opts)
		}

		cl, err = client.New(cfg, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to construct the kubernetes client: %w", err)
		}

		if cf.MaxClientCacheTTL == 0 {
			cf.MaxClientCacheTTL = DefaultMaxClientCacheTTL
		}

		// if the config getter didn't give us a ttl or it is too large, reset it to the max
		if ttl == 0 || ttl > cf.MaxClientCacheTTL {
			ttl = cf.MaxClientCacheTTL
		}

		cf.cache.Set(key, cl, ttl)
	}

	return cl.(client.Client), nil
}

type restConfigGetter interface {
	GetCacheKey(ctx context.Context) (cacheKey, error)
	GetRestConfig(ctx context.Context) (*rest.Config, time.Duration, error)
}

type kubeConfigRestConfigGetter struct {
	CurrentNamespace     string
	ApiUrl               string
	KubeConfigSecretName string
	Client               client.Client

	kubeConfigData []byte
}

var _ restConfigGetter = (*kubeConfigRestConfigGetter)(nil)

// GetCacheKey implements restConfigGetter
func (g *kubeConfigRestConfigGetter) GetCacheKey(ctx context.Context) (key cacheKey, err error) {
	if err = g.ensureKubeConfig(ctx); err != nil {
		return
	}

	sum := sha256.Sum256(g.kubeConfigData)
	key.kubeConfigHash = string(sum[:])
	return
}

// GetRestConfig implements restConfigGetter
func (g *kubeConfigRestConfigGetter) GetRestConfig(ctx context.Context) (cfg *rest.Config, ttl time.Duration, err error) {
	if err = g.ensureKubeConfig(ctx); err != nil {
		return
	}

	cfg, err = clientcmd.BuildConfigFromKubeconfigGetter(g.ApiUrl, clientcmd.KubeconfigGetter(func() (*clientcmdapi.Config, error) {
		return clientcmd.Load(g.kubeConfigData) //nolint:wrapcheck // this is handled byt the outer error
	}))
	if err != nil {
		err = fmt.Errorf("failed to process kubeconfig in secret '%s': %w", g.KubeConfigSecretName, err)
	}

	return
}

func (g *kubeConfigRestConfigGetter) ensureKubeConfig(ctx context.Context) error {
	sec := &corev1.Secret{}
	if err := g.Client.Get(ctx, client.ObjectKey{Name: g.KubeConfigSecretName, Namespace: g.CurrentNamespace}, sec); err != nil {
		return fmt.Errorf("failed to get the secret with the kubeconfig: %w", err)
	}
	g.kubeConfigData = sec.Data["kubeconfig"]
	return nil
}

type inNamespaceServiceAccountRestConfigGetter struct {
	CurrentNamespace string
	Client           client.Client
	Config           *rest.Config

	sa *corev1.ServiceAccount
}

var _ restConfigGetter = (*inNamespaceServiceAccountRestConfigGetter)(nil)

// GetCacheKey implements restConfigGetter
func (g *inNamespaceServiceAccountRestConfigGetter) GetCacheKey(ctx context.Context) (key cacheKey, err error) {
	if err = g.ensureSA(ctx); err != nil {
		return
	}

	key.serviceAccountKey = client.ObjectKeyFromObject(g.sa)
	return
}

// GetRestConfig implements restConfigGetter
func (g *inNamespaceServiceAccountRestConfigGetter) GetRestConfig(ctx context.Context) (cfg *rest.Config, ttl time.Duration, err error) {
	if err = g.ensureSA(ctx); err != nil {
		return
	}

	tr := &auth.TokenRequest{}

	if err := g.Client.SubResource("token").Create(ctx, g.sa, tr); err != nil {
		return nil, ttl, fmt.Errorf("failed to obtain the token for service account '%s': %w", g.sa.Name, err)
	}

	cfg = rest.CopyConfig(g.Config)
	cfg.BearerToken = tr.Status.Token
	// invalidate all other means of authentication
	cfg.TLSClientConfig.CertData = nil
	cfg.TLSClientConfig.CertFile = ""
	cfg.Username = ""
	cfg.Password = ""
	cfg.Impersonate = rest.ImpersonationConfig{}

	ttl = time.Until(tr.Status.ExpirationTimestamp.Time)

	return
}

func (g *inNamespaceServiceAccountRestConfigGetter) ensureSA(ctx context.Context) error {
	if g.sa != nil {
		return nil
	}

	list := &corev1.ServiceAccountList{}
	if err := g.Client.List(ctx, list, client.InNamespace(g.CurrentNamespace), client.HasLabels{RemoteSecretAuthServiceAccountLabel}); err != nil {
		return fmt.Errorf("failed to list the potential SAs to use for remote secret deployment: %w", err)
	}

	if len(list.Items) == 0 {
		return noAuthServiceAccountFound
	}

	if len(list.Items) > 1 {
		return onlyOneAuthServiceAccountPerNamespaceAllowed
	}

	g.sa = &list.Items[0]

	return nil
}
