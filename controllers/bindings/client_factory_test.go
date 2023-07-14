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
	"testing"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/stretchr/testify/assert"
	auth "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestGetClient(t *testing.T) {
	t.Run("with kubeconfig secret", func(t *testing.T) {
		cl := fake.NewClientBuilder().
			WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "ns",
					},
					Data: map[string][]byte{
						"kubeconfig": []byte(`
apiVersion: v1
kind: Config
clusters: []
contexts: []
users: []
preferences: {}
`),
					},
				},
			).
			Build()

		cf := CachingClientFactory{
			LocalCluster: LocalClusterConnectionDetails{
				Client: cl,
				Config: &rest.Config{
					Host: "api.host",
				},
			},
		}

		tcl, err := cf.GetClient(context.TODO(), "ns", &api.RemoteSecretTarget{
			Namespace:                "ns2",
			ClusterCredentialsSecret: "secret",
		}, nil)

		assert.NoError(t, err)
		assert.NotNil(t, tcl)
	})

	t.Run("uses SA", func(t *testing.T) {
		cl := fake.NewClientBuilder().
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceCreate: func(ctx context.Context, client client.Client, subResourceName string, obj, subResource client.Object, opts ...client.SubResourceCreateOption) error {
					if _, ok := obj.(*corev1.ServiceAccount); ok && subResourceName == "token" && obj.GetName() == "auth-sa" {
						tr := subResource.(*auth.TokenRequest)
						tr.Status.Token = "le-token"
					}
					return nil
				},
			}).
			WithObjects(
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "auth-sa",
						Namespace: "ns",
						Labels: map[string]string{
							RemoteSecretAuthServiceAccountLabel: "true",
						},
					},
				},
			).
			Build()

		cf := CachingClientFactory{
			LocalCluster: LocalClusterConnectionDetails{
				Client: cl,
				Config: &rest.Config{
					Host: "api.host",
				},
			},
		}

		tcl, err := cf.GetClient(context.TODO(), "ns", nil, &api.TargetStatus{
			Namespace: "ns2",
		})

		assert.NoError(t, err)
		assert.NotNil(t, tcl)
	})

	t.Run("refuses to work without SA", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		cf := CachingClientFactory{
			LocalCluster: LocalClusterConnectionDetails{
				Client: cl,
				Config: &rest.Config{
					Host: "api.host",
				},
			},
		}

		for _, tt := range []struct {
			name      string
			useStatus bool
		}{
			{
				name:      "with spec",
				useStatus: false,
			},
			{
				name:      "with status",
				useStatus: true,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				var spec *api.RemoteSecretTarget
				var status *api.TargetStatus
				if tt.useStatus {
					status = &api.TargetStatus{
						Namespace: "other_ns",
					}
				} else {
					spec = &api.RemoteSecretTarget{
						Namespace: "other_ns",
					}
				}

				tcl, err := cf.GetClient(context.TODO(), "ns", spec, status)

				assert.Error(t, err)
				assert.Nil(t, tcl)
			})
		}
	})

	t.Run("uses the provided client for the current namespace", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		cf := CachingClientFactory{
			LocalCluster: LocalClusterConnectionDetails{
				Client: cl,
				Config: &rest.Config{
					Host: "api.host",
				},
			},
		}

		for _, tt := range []struct {
			name      string
			url       string
			useStatus bool
		}{
			{
				name:      "with spec with explicit API URL",
				url:       "api.host",
				useStatus: false,
			},
			{
				name:      "with spec without explicit API URL",
				url:       "",
				useStatus: false,
			},
			{
				name:      "with status with explicit API URL",
				url:       "api.host",
				useStatus: true,
			},
			{
				name:      "with status without explicit API URL",
				url:       "",
				useStatus: true,
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				var spec *api.RemoteSecretTarget
				var status *api.TargetStatus
				if tt.useStatus {
					status = &api.TargetStatus{
						Namespace: "ns",
						ApiUrl:    tt.url,
					}
				} else {
					spec = &api.RemoteSecretTarget{
						Namespace: "ns",
						ApiUrl:    tt.url,
					}
				}

				tcl, err := cf.GetClient(context.TODO(), "ns", spec, status)

				assert.NoError(t, err)
				assert.Same(t, cl, tcl)
			})
		}
	})

	t.Run("uses cache", func(t *testing.T) {})
	t.Run("evicts cache after timeout", func(t *testing.T) {})
}

func TestKubeConfigRestConfigGetter(t *testing.T) {
	kubeConfig := []byte(`
apiVersion: v1
kind: Config
clusters: []
contexts: []
users: []
preferences: {}
`)
	cl := fake.NewClientBuilder().
		WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "ns",
				},
				Data: map[string][]byte{
					"kubeconfig": kubeConfig,
				},
			},
		).
		Build()

	getter := kubeConfigRestConfigGetter{
		CurrentNamespace:     "ns",
		ApiUrl:               "api.host",
		KubeConfigSecretName: "secret",
		Client:               cl,
	}

	key, err := getter.GetCacheKey(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, kubeConfig, getter.kubeConfigData)

	sum := sha256.Sum256(kubeConfig)
	assert.Equal(t, string(sum[:]), key.kubeConfigHash)
	assert.Empty(t, key.serviceAccountKey.Name)
	assert.Empty(t, key.serviceAccountKey.Namespace)

	cfg, _, err := getter.GetRestConfig(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestInNamespaceServiceAccountRestConfigGetter(t *testing.T) {
	cl := fake.NewClientBuilder().
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceCreate: func(ctx context.Context, client client.Client, subResourceName string, obj, subResource client.Object, opts ...client.SubResourceCreateOption) error {
				if _, ok := obj.(*corev1.ServiceAccount); ok && subResourceName == "token" && obj.GetName() == "auth-sa" {
					tr := subResource.(*auth.TokenRequest)
					tr.Status.Token = "le-token"
				}
				return nil
			},
		}).
		WithObjects(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "auth-sa",
					Namespace: "ns",
					Labels: map[string]string{
						RemoteSecretAuthServiceAccountLabel: "true",
					},
				},
			},
		).
		Build()

	getter := inNamespaceServiceAccountRestConfigGetter{
		CurrentNamespace: "ns",
		Client:           cl,
		Config: &rest.Config{
			Host: "api.host",
		},
	}

	key, err := getter.GetCacheKey(context.TODO())
	assert.NoError(t, err)
	assert.Empty(t, key.kubeConfigHash)
	assert.Equal(t, "auth-sa", key.serviceAccountKey.Name)
	assert.Equal(t, "ns", key.serviceAccountKey.Namespace)

	cfg, _, err := getter.GetRestConfig(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, "le-token", cfg.BearerToken)
}
