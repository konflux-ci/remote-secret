package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateRemoteSecret(t *testing.T) {
	
	TUr := TokenUploadReconciler{
		Client:              fake.NewClientBuilder().Build(),
		Scheme:              nil,
		RemoteSecretStorage: rss,
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
