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

func TestCheckMatchingSecretTypes(t *testing.T) {
	test := func(rsType, uploadType corev1.SecretType, shouldError bool) {
		err := checkMatchingSecretTypes(rsType, uploadType)
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
		test(corev1.SSHAuthPrivateKey, corev1.SSHAuthPrivateKey, false)
	})

	t.Run("types do not match", func(t *testing.T) {
		test("", corev1.SecretTypeDockercfg, true)
		test(corev1.SecretTypeDockercfg, corev1.SecretTypeOpaque, true)
		test(corev1.SecretTypeBasicAuth, corev1.SecretTypeSSHAuth, true)
		test("rick", "morty", true)
	})
}

func TestValidateUploadSecret(t *testing.T) {
	rs := &RemoteSecret{
		Spec: RemoteSecretSpec{
			Secret: LinkableSecretSpec{
				Type:         corev1.SecretTypeSSHAuth,
				RequiredKeys: []SecretKey{{"foo"}, {"bar"}},
			},
		},
	}
	upload := &corev1.Secret{
		Type: corev1.SecretTypeSSHAuth,
		Data: map[string][]byte{
			corev1.SSHAuthPrivateKey: []byte("ssh-key...."),
			"foo":                    []byte("whatever"),
			"bar":                    []byte("forever"),
		},
	}

	t.Run("types match and contains all required keys", func(t *testing.T) {
		err := rs.ValidateUploadSecret(upload)
		assert.NoError(t, err)
	})

	t.Run("types do not match", func(t *testing.T) {
		myUpload := upload.DeepCopy()
		myUpload.Type = corev1.SecretTypeOpaque
		err := rs.ValidateUploadSecret(myUpload)
		assert.Error(t, err)
	})

	t.Run("required keys from spec are not in secret", func(t *testing.T) {
		myRS := rs.DeepCopy()
		myRS.Spec.Secret.RequiredKeys = []SecretKey{{"scoby-dooo"}}
		err := myRS.ValidateUploadSecret(upload)
		assert.Error(t, err)
	})

	t.Run("required keys for type not in secret", func(t *testing.T) { // This should not happen in real cluster.
		myUpload := upload.DeepCopy()
		delete(myUpload.Data, corev1.SSHAuthPrivateKey)
		err := rs.ValidateUploadSecret(myUpload)
		assert.Error(t, err)
	})
}

func TestValidateSecretData(t *testing.T) {
	t.Run("does not contain key from RemoteSecret spec", func(t *testing.T) {
		rs := RemoteSecret{Spec: RemoteSecretSpec{Secret: LinkableSecretSpec{
			Type:         corev1.SecretTypeDockercfg,
			RequiredKeys: []SecretKey{{Name: "foo"}, {Name: "bar"}},
		}}}
		secretData := map[string][]byte{"foo": []byte("whatever"), "kuku": []byte("other")}

		err := rs.ValidateSecretData(secretData)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "bar")
		assert.ErrorContains(t, err, corev1.DockerConfigKey)
		assert.NotContains(t, err.Error(), "foo")
		assert.NotContains(t, err.Error(), "kuku")
	})

	t.Run("contains only one key from two required keys for TLS type", func(t *testing.T) {
		rs := RemoteSecret{Spec: RemoteSecretSpec{Secret: LinkableSecretSpec{
			Type:         corev1.SecretTypeTLS,
			RequiredKeys: []SecretKey{{Name: "foo"}},
		}}}
		secretData := map[string][]byte{"foo": []byte("whatever"), corev1.TLSCertKey: []byte("tlscert...")}

		err := rs.ValidateSecretData(secretData)
		assert.Error(t, err)
		assert.ErrorContains(t, err, corev1.TLSPrivateKeyKey)
	})

	t.Run("contains both keys for TLS type", func(t *testing.T) {
		rs := RemoteSecret{Spec: RemoteSecretSpec{Secret: LinkableSecretSpec{
			Type: corev1.SecretTypeTLS,
		}}}
		secretData := map[string][]byte{corev1.TLSCertKey: []byte("tlscert..."), corev1.TLSPrivateKeyKey: []byte("tlskey...")}

		err := rs.ValidateSecretData(secretData)
		assert.NoError(t, err)
	})

	t.Run("does not contain at least on key for basic-auth type", func(t *testing.T) {
		rs := RemoteSecret{Spec: RemoteSecretSpec{Secret: LinkableSecretSpec{
			Type: corev1.SecretTypeBasicAuth,
		}}}
		secretData := map[string][]byte{"foo": []byte("whatever")}

		err := rs.ValidateSecretData(secretData)
		assert.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("%s neither %s", corev1.BasicAuthUsernameKey, corev1.BasicAuthPasswordKey))
	})

	t.Run("contains at least on key for basic-auth type", func(t *testing.T) {
		rs := RemoteSecret{Spec: RemoteSecretSpec{Secret: LinkableSecretSpec{
			Type: corev1.SecretTypeBasicAuth,
		}}}
		secretData := map[string][]byte{corev1.BasicAuthUsernameKey: []byte("user")}

		err := rs.ValidateSecretData(secretData)
		assert.NoError(t, err)
	})

}
