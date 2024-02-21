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

	remoteSecret := &api.RemoteSecret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "appstudio.redhat.com/v1beta1",
			Kind:       "RemoteSecret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-remote-secret",
		},
	}

	rs_gvc := schema.GroupVersionKind{
		Group:   "appstudio.redhat.com",
		Version: "v1beta",
		Kind:    "RemoteSecret",
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(rs_gvc, &api.RemoteSecret{})
	assert.NoError(t, corev1.AddToScheme(scheme))
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(remoteSecret).Build()

	TUr := TokenUploadReconciler{
		Client:              cl,
		Scheme:              nil,
		RemoteSecretStorage: nil,
	}

	// Define the secret here to avoid repetition and only overwrite the specific parts in each test case.
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
	rs1, err := TUr.createRemoteSecret(context.TODO(), uploadSecret)
	assert.NoError(t, err)
	assert.NotNil(t, rs1)

	// Second create should return the same remote secret
	rs2, err2 := TUr.createRemoteSecret(context.TODO(), uploadSecret)
	assert.NoError(t, err2)
	assert.NotNil(t, rs2)

	assert.Equal(t, *rs1, *rs2)
}
