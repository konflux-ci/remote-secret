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

package remotesecretstorage

import (
	"context"
	"testing"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage/memorystorage"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPartialUpdate(t *testing.T) {
	rss := remoteSecretStorage{
		DefaultTypedSecretStorage: secretstorage.DefaultTypedSecretStorage[api.RemoteSecret, SecretData]{
			DataTypeName:  "remote secret",
			SecretStorage: &memorystorage.MemoryStorage{},
			ToID:          secretstorage.ObjectToID[*api.RemoteSecret],
			Serialize:     secretstorage.SerializeJSON[SecretData],
			Deserialize:   secretstorage.DeserializeJSON[SecretData],
		},
	}
	id := &api.RemoteSecret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			UID:       "asdf",
		},
	}
	before := func(t *testing.T) {
		assert.NoError(t, rss.Store(context.TODO(), id, &SecretData{
			"k1": []byte("v1"),
			"k2": []byte("v2"),
		}))
	}

	t.Run("updates", func(t *testing.T) {
		before(t)
		err := rss.PartialUpdate(context.TODO(), id, &SecretData{
			"k2": []byte("v2_new"),
			"k3": []byte("v3"),
		}, nil)

		assert.NoError(t, err)

		updated, err := rss.Get(context.TODO(), id)
		assert.NoError(t, err)

		assert.Len(t, *updated, 3)
		assert.Equal(t, (*updated)["k1"], []byte("v1"))
		assert.Equal(t, (*updated)["k2"], []byte("v2_new"))
		assert.Equal(t, (*updated)["k3"], []byte("v3"))
	})

	t.Run("deletes", func(t *testing.T) {
		before(t)
		err := rss.PartialUpdate(context.TODO(), id, nil, []string{"k2", "nonexistent"})

		assert.NoError(t, err)

		updated, err := rss.Get(context.TODO(), id)
		assert.NoError(t, err)

		assert.Len(t, *updated, 1)
		assert.Equal(t, (*updated)["k1"], []byte("v1"))
	})
}
