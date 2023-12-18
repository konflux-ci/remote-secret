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

package bindings

import (
	"context"
	"testing"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSync(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))

	clBld := func() *fake.ClientBuilder {
		return fake.NewClientBuilder().WithScheme(scheme)
	}

	token := &api.RemoteSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token",
			Namespace: "default",
		},
	}

	deploymentTarget := &TestDeploymentTarget{}
	secretBuilder := &TestSecretDataGetter[*api.RemoteSecret]{}
	h := secretHandler[*api.RemoteSecret]{
		Target:           deploymentTarget,
		ObjectMarker:     &TestObjectMarker{},
		SecretDataGetter: secretBuilder,
	}

	t.Run("empty-cluster", func(t *testing.T) {
		t.Run("service-account-token secret type", func(t *testing.T) {
			cl := clBld().Build()
			deploymentTarget.GetSpecImpl = func() api.LinkableSecretSpec {
				return api.LinkableSecretSpec{
					Name: "secret",
					Type: corev1.SecretTypeServiceAccountToken,
				}
			}
			deploymentTarget.GetClientImpl = func() client.Client { return cl }
			deploymentTarget.GetTargetNamespaceImpl = func() string {
				return "ns"
			}

			secretBuilder.GetDataImpl = func(ctx context.Context, st *api.RemoteSecret) (map[string][]byte, string, error) {
				return map[string][]byte{
					"extra": []byte("token"),
				}, "", nil
			}

			secret, reason, err := h.Sync(context.TODO(), token, false)
			assert.Equal(t, "", reason)
			assert.NoError(t, err)

			assert.NotNil(t, secret)
			assert.Contains(t, secret.Data, "extra")
			assert.Equal(t, secret.Data["extra"], []byte("token"))
		})

		t.Run("other secret types", func(t *testing.T) {
			cl := clBld().Build()
			deploymentTarget.GetSpecImpl = func() api.LinkableSecretSpec {
				return api.LinkableSecretSpec{
					Name: "secret",
					Type: corev1.SecretTypeBasicAuth,
				}
			}
			deploymentTarget.GetClientImpl = func() client.Client { return cl }
			deploymentTarget.GetTargetNamespaceImpl = func() string {
				return "ns"
			}

			secretBuilder.GetDataImpl = func(ctx context.Context, st *api.RemoteSecret) (map[string][]byte, string, error) {
				return map[string][]byte{
					"token": []byte("token"),
				}, "", nil
			}

			secret, reason, err := h.Sync(context.TODO(), token, false)
			assert.Equal(t, "", reason)
			assert.NoError(t, err)

			assert.NotNil(t, secret)
			assert.Contains(t, secret.Data, "token")
			assert.Equal(t, secret.Data["token"], []byte("token"))
		})
	})

	t.Run("secret-in-cluster", func(t *testing.T) {
		t.Run("service-account-token secret type", func(t *testing.T) {
			cl := clBld().
				WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "ns",
					},
					Data: map[string][]byte{
						"a": []byte("b"),
					},
				}).
				Build()

			deploymentTarget.GetSpecImpl = func() api.LinkableSecretSpec {
				return api.LinkableSecretSpec{
					Name: "secret",
					Type: corev1.SecretTypeServiceAccountToken,
				}
			}
			deploymentTarget.GetClientImpl = func() client.Client {
				return cl
			}
			deploymentTarget.GetTargetNamespaceImpl = func() string {
				return "ns"
			}

			secretBuilder.GetDataImpl = func(ctx context.Context, st *api.RemoteSecret) (map[string][]byte, string, error) {
				return map[string][]byte{
					"extra": []byte("token"),
				}, "", nil
			}

			secret, reason, err := h.Sync(context.TODO(), token, false)
			assert.Equal(t, "", reason)
			assert.NoError(t, err)

			assert.NotNil(t, secret)
			assert.Contains(t, secret.Data, "extra")
			assert.Equal(t, secret.Data["extra"], []byte("token"))
			assert.NotContains(t, secret.Data, "a")
		})

		t.Run("other secret types", func(t *testing.T) {
			cl := clBld().
				WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "ns",
					},
					Data: map[string][]byte{
						"a": []byte("b"),
					},
				}).
				Build()

			deploymentTarget.GetSpecImpl = func() api.LinkableSecretSpec {
				return api.LinkableSecretSpec{
					Name: "secret",
					Type: corev1.SecretTypeBasicAuth,
				}
			}
			deploymentTarget.GetClientImpl = func() client.Client {
				return cl
			}
			deploymentTarget.GetTargetNamespaceImpl = func() string {
				return "ns"
			}

			secretBuilder.GetDataImpl = func(ctx context.Context, st *api.RemoteSecret) (map[string][]byte, string, error) {
				return map[string][]byte{
					"token": []byte("token"),
				}, "", nil
			}

			secret, reason, err := h.Sync(context.TODO(), token, false)
			assert.Equal(t, "", reason)
			assert.NoError(t, err)

			assert.NotNil(t, secret)
			assert.Contains(t, secret.Data, "token")
			assert.Equal(t, secret.Data["token"], []byte("token"))
			assert.NotContains(t, secret.Data, "a")
		})
		t.Run("honors recreate flag", func(t *testing.T) {
			cl := clBld().
				WithObjects(&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret",
						Namespace: "ns",
					},
				}).
				Build()

			deploymentTarget.GetSpecImpl = func() api.LinkableSecretSpec {
				return api.LinkableSecretSpec{
					Name: "secret",
				}
			}
			deploymentTarget.GetClientImpl = func() client.Client {
				return cl
			}
			deploymentTarget.GetTargetNamespaceImpl = func() string {
				return "ns"
			}
			deploymentTarget.GetActualSecretNameImpl = func() string {
				return "old-secret"
			}

			secret, reason, err := h.Sync(context.TODO(), token, true)
			assert.Equal(t, "", reason)
			assert.NoError(t, err)

			assert.NotNil(t, secret)
			assert.Equal(t, "secret", secret.Name)
		})
	})
}

func TestList(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a",
					Namespace: "default",
					Labels: map[string]string{
						"thus-laybeled": "you-should-be, but you're not",
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "b",
					Namespace: "default",
					Labels: map[string]string{
						"not-labeled": "correctly",
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: "different-one",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shes-the-one",
					Namespace: "default",
					Labels: map[string]string{
						"thus-laybeled": "you-should-be",
					},
				},
			},
		).
		Build()

	h := secretHandler[*api.RemoteSecret]{
		Target: &TestDeploymentTarget{
			GetClientImpl: func() client.Client { return cl },
		},
		ObjectMarker: &TestObjectMarker{
			ListManagedOptionsImpl: func(ctx context.Context, _ client.ObjectKey) ([]client.ListOption, error) {
				return []client.ListOption{
					client.MatchingLabels{"thus-laybeled": "you-should-be"},
				}, nil
			},
			IsManagedByImpl: func(ctx context.Context, _ client.ObjectKey, o client.Object) (bool, error) {
				return o.GetLabels()["thus-laybeled"] == "you-should-be", nil
			},
		},
		SecretDataGetter: &TestSecretDataGetter[*api.RemoteSecret]{},
	}

	scs, err := h.List(context.TODO())
	assert.NoError(t, err)

	assert.Len(t, scs, 1)
	assert.Equal(t, scs[0].Name, "shes-the-one")
}

func TestCheckColliding(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, corev1.AddToScheme(scheme))

	t.Run("Secret does not exist", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(scheme).Build()
		h := secretHandler[*api.RemoteSecret]{
			Target: &TestDeploymentTarget{
				GetClientImpl: func() client.Client { return cl },
			},
			ObjectMarker:     &TestObjectMarker{},
			SecretDataGetter: &TestSecretDataGetter[*api.RemoteSecret]{},
		}

		err := h.CheckColliding(context.TODO())
		assert.NoError(t, err)
	})

	t.Run("Secret is managed by other RemoteSecret", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "managed-by-other",
					Namespace: "default",
				},
			}).Build()
		h := secretHandler[*api.RemoteSecret]{
			Target: &TestDeploymentTarget{
				GetClientImpl: func() client.Client { return cl },
				GetTargetNamespaceImpl: func() string {
					return "default"
				},
				GetSpecImpl: func() api.LinkableSecretSpec {
					return api.LinkableSecretSpec{
						Name: "managed-by-other",
					}
				},
			},
			ObjectMarker: &TestObjectMarker{
				IsManagedByOtherImpl: func(ctx context.Context, _ client.ObjectKey, o client.Object) (bool, client.ObjectKey, error) {
					return true, client.ObjectKey{}, nil
				},
			},
			SecretDataGetter: &TestSecretDataGetter[*api.RemoteSecret]{},
		}

		err := h.CheckColliding(context.TODO())
		assert.Error(t, err)
		assert.ErrorContains(t, err, "other")
	})
}
