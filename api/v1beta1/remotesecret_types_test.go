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

package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateUploadSecret(t *testing.T) {
	test := func(rsType, uploadType corev1.SecretType, shouldError bool) {
		remoteSecret := &RemoteSecret{
			Spec: RemoteSecretSpec{
				Secret: LinkableSecretSpec{
					Type: rsType,
				},
			},
		}
		uploadSecret := &corev1.Secret{
			Type: uploadType,
		}
		err := remoteSecret.ValidateUploadSecret(uploadSecret)
		if shouldError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}

	t.Run("types match", func(t *testing.T) {
		test("", "", false)
		test("", corev1.SecretTypeOpaque, false)
		test(corev1.SecretTypeOpaque, "", false)
		test("foo", "foo", false)
	})

	t.Run("types do not match", func(t *testing.T) {
		test("", corev1.SecretTypeDockercfg, true)
		test(corev1.SecretTypeDockercfg, corev1.SecretTypeOpaque, true)
		test(corev1.SecretTypeBasicAuth, corev1.SecretTypeSSHAuth, true)
		test("rick", "morty", true)
	})

	t.Run("types do not match", func(t *testing.T) {
		rs := &RemoteSecret{
			Spec: RemoteSecretSpec{
				Secret: LinkableSecretSpec{
					Type:         corev1.SecretTypeSSHAuth,
					RequiredKeys: []SecretKey{{"foo"}, {"bar"}},
				},
			},
		}
		uploadSecret := &corev1.Secret{
			Type: corev1.SecretTypeSSHAuth,
			Data: map[string][]byte{
				corev1.SSHAuthPrivateKey: []byte("ssh-key...."),
			},
		}
	})

}

func TestValidateSecretData(t *testing.T) {
	rs := RemoteSecret{
		Spec: RemoteSecretSpec{
			Secret: LinkableSecretSpec{
				Type:         corev1.SecretTypeDockercfg,
				RequiredKeys: []SecretKey{{Name: "foo"}, {Name: "bar"}},
			},
		},
	}
	secretData := map[string][]byte{
		"foo":  []byte("whatever"),
		"kuku": []byte("other"),
	}

	t.Run("basic scenario", func(t *testing.T) {
		rs := RemoteSecret{
			Spec: RemoteSecretSpec{
				Secret: LinkableSecretSpec{
					Type:         corev1.SecretTypeDockercfg,
					RequiredKeys: []SecretKey{{Name: "foo"}, {Name: "bar"}},
				},
			},
		}
		secretData := map[string][]byte{
			"foo":  []byte("whatever"),
			"kuku": []byte("other"),
		}

		err := rs.ValidateSecretData(secretData)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "bar")
		assert.ErrorContains(t, err, corev1.DockerConfigKey)
		assert.NotContains(t, err.Error(), "foo")
		assert.NotContains(t, err.Error(), "kuku")
	})

	t.Run("contains only one key from two required keys for TLS type", func(t *testing.T) {
		rs := RemoteSecret{
			Spec: RemoteSecretSpec{
				Secret: LinkableSecretSpec{
					Type:         corev1.SecretTypeTLS,
					RequiredKeys: []SecretKey{{Name: "foo"}},
				},
			},
		}
		secretData := map[string][]byte{
			"foo":             []byte("whatever"),
			corev1.TLSCertKey: []byte("tlscert..."),
		}

		err := rs.ValidateSecretData(secretData)
		assert.Error(t, err)
		assert.ErrorContains(t, err, corev1.TLSPrivateKeyKey)
	})

	t.Run("does not contain at least on key for basic-auth type", func(t *testing.T) {
		rs := RemoteSecret{
			Spec: RemoteSecretSpec{
				Secret: LinkableSecretSpec{
					Type:         corev1.SecretTypeBasicAuth,
					RequiredKeys: []SecretKey{{Name: "foo"}},
				},
			},
		}
		secretData := map[string][]byte{
			"foo": []byte("whatever"),
		}

		err := rs.ValidateSecretData(secretData)
		assert.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("%s neither %s", corev1.BasicAuthUsernameKey, corev1.BasicAuthPasswordKey))
	})

}
