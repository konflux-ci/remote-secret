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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateUploadSecretType(t *testing.T) {
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
		err := remoteSecret.ValidateUploadSecretType(uploadSecret)
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

}
