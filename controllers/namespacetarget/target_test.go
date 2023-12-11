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

package namespacetarget

import (
	"testing"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNamespaceTarget_GetActualSecretName(t *testing.T) {
	bt := getTestNamespaceTarget()
	assert.Equal(t, "kachny-asdf", bt.GetActualSecretName())
}

func TestNamespaceTarget_GetActualServiceAccountNames(t *testing.T) {
	bt := getTestNamespaceTarget()

	assert.Equal(t, []string{"a", "b"}, bt.GetActualServiceAccountNames())
}

func TestNamespaceTarget_GetClient(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	bt := getTestNamespaceTarget()
	bt.Client = cl

	assert.Same(t, cl, bt.GetClient())
}

func TestNamespaceTarget_GetTargetObjectKey(t *testing.T) {
	bt := getTestNamespaceTarget()
	assert.Equal(t, client.ObjectKey{Name: "remotesecret", Namespace: "ns"}, bt.GetTargetObjectKey())
}

func TestNamespaceTarget_GetSpec(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		bt := getTestNamespaceTarget()

		assert.Equal(t, api.LinkableSecretSpec{
			GenerateName: "kachny-",
		}, bt.GetSpec())
	})

	t.Run("with overrides", func(t *testing.T) {
		nt := getNamespaceTargetFrom(getTestRemoteSecretWithOverrides())

		assert.Equal(t, api.LinkableSecretSpec{
			Name:         "target-secret",
			GenerateName: "kachny-",
			Labels: map[string]string{
				"k": "v",
			},
			Annotations: map[string]string{
				"k": "v",
			},
		}, nt.GetSpec())
	})
}

func TestNamespaceTarget_GetTargetNamespace(t *testing.T) {
	bt := getTestNamespaceTarget()
	assert.Equal(t, "target-ns", bt.GetTargetNamespace())
}

func TestNamespaceTarget_GetType(t *testing.T) {
	assert.Equal(t, "Namespace", (&NamespaceTarget{}).GetType())
}

func getTestNamespaceTarget() NamespaceTarget {
	return getNamespaceTargetFrom(getTestRemoteSecret())
}

func getNamespaceTargetFrom(rs *api.RemoteSecret) NamespaceTarget {
	return NamespaceTarget{
		Client:       nil,
		TargetKey:    client.ObjectKeyFromObject(rs),
		SecretSpec:   &rs.Spec.Secret,
		TargetSpec:   &rs.Spec.Targets[0],
		TargetStatus: &rs.Status.Targets[0],
	}
}

func getTestRemoteSecret() *api.RemoteSecret {
	return &api.RemoteSecret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "remotesecret",
			Namespace: "ns",
		},
		Spec: api.RemoteSecretSpec{
			Secret: api.LinkableSecretSpec{
				GenerateName: "kachny-",
			},
			Targets: []api.RemoteSecretTarget{
				{
					Namespace: "target-ns",
				},
			},
		},
		Status: api.RemoteSecretStatus{
			Targets: []api.TargetStatus{
				{
					Namespace: "target-ns",
					DeployedSecret: &api.DeployedSecretStatus{
						Name: "kachny-asdf",
					},
					ServiceAccountNames: []string{"a", "b"},
				},
			},
		},
	}
}

func getTestRemoteSecretWithOverrides() *api.RemoteSecret {
	return &api.RemoteSecret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "remotesecret",
			Namespace: "ns",
		},
		Spec: api.RemoteSecretSpec{
			Secret: api.LinkableSecretSpec{
				GenerateName: "kachny-",
			},
			Targets: []api.RemoteSecretTarget{
				{
					Namespace: "target-ns",
					Secret: &api.SecretOverride{
						Name: "target-secret",
						Labels: &map[string]string{
							"k": "v",
						},
						Annotations: &map[string]string{
							"k": "v",
						},
					},
				},
			},
		},
		Status: api.RemoteSecretStatus{
			Targets: []api.TargetStatus{
				{
					Namespace: "target-ns",
					DeployedSecret: &api.DeployedSecretStatus{
						Name: "target-secret",
					},
					ServiceAccountNames: []string{"a", "b"},
				},
			},
		},
	}
}
