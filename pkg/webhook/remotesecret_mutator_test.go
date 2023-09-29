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
