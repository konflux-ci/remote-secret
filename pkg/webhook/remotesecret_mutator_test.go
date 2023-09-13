package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/memorystorage"
)

func TestStoreUploadData(t *testing.T) {
	rs := &api.RemoteSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs",
			Namespace: "ns",
		},
		UploadData: map[string]string{
			"a": "b",
		},
	}

	storage := remotesecretstorage.NewJSONSerializingRemoteSecretStorage(&memorystorage.MemoryStorage{})

	m := RemoteSecretMutator{
		Client:  nil,
		Storage: storage,
	}

	assert.NoError(t, m.StoreUploadData(context.TODO(), rs))

	data, err := storage.Get(context.TODO(), rs)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Len(t, *data, 1)
	assert.Equal(t, []byte("b"), (*data)["a"])
}

func TestStoreCopyDataFrom(t *testing.T) {
	t.Skip("not testable until we use controller-runtime >= 0.15.x because we need to fake SubjectAccessReview using interceptors")
}
