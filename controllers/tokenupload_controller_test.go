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

package controllers

import (
	"context"
	"testing"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateRemoteSecret(t *testing.T) {

	// Create client with RemoteSecret in its scheme
	rsGVK := schema.GroupVersionKind{
		Group:   "appstudio.redhat.com",
		Version: "v1beta",
		Kind:    "RemoteSecret",
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(rsGVK, &api.RemoteSecret{})
	assert.NoError(t, corev1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	TUr := TokenUploadReconciler{
		Client:              cl,
		Scheme:              nil,
		RemoteSecretStorage: nil,
	}

	// Define the upload secret
	uploadSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-remote-secret-upload",
			Namespace: "default",
			Labels: map[string]string{
				"appstudio.redhat.com/upload-secret": "remotesecret",
			},
			Annotations: map[string]string{
				"appstudio.redhat.com/remotesecret-name": "test-remote-secret",
			},
		},
	}

	// check remote secret does not exist
	rs, err := TUr.findRemoteSecret(context.TODO(), uploadSecret)
	assert.Nil(t, rs)
	assert.NoError(t, err)

	// create remote secret
	rs1, err1 := TUr.createRemoteSecret(context.TODO(), uploadSecret)
	assert.NotNil(t, rs1)
	assert.NoError(t, err1)

	// check remote secret exists
	rs2, err2 := TUr.findRemoteSecret(context.TODO(), uploadSecret)
	assert.NotNil(t, rs2)
	assert.NoError(t, err2)

	// Second create should return the same remote secret
	rs3, err3 := TUr.createRemoteSecret(context.TODO(), uploadSecret)
	assert.NoError(t, err3)
	assert.NotNil(t, rs3)
	assert.Equal(t, rs1.Name, rs3.Name)
}
