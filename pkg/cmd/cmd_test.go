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

package cmd

import (
	"context"
	"testing"

	"github.com/redhat-appstudio/remote-secret/pkg/config"

	"github.com/stretchr/testify/assert"
)

func TestCreateTokenStorage(t *testing.T) {
	err := config.SetupCustomValidations(config.CustomValidationOptions{AllowInsecureURLs: true})
	assert.NoError(t, err)

	t.Run("unsupported type", func(t *testing.T) {
		var blabol TokenStorageType = "eh"

		ss, err := CreateInitializedSecretStorage(context.TODO(), nil, &CommonCliArgs{TokenStorage: blabol})

		assert.Nil(t, ss)
		assert.Error(t, err)
		assert.ErrorIs(t, err, errUnsupportedSecretStorage)
	})

	t.Run("fail AWS new", func(t *testing.T) {
		// this fails, because it lacks the proper AWS client configuration
		ss, err := CreateInitializedSecretStorage(context.TODO(), nil, &CommonCliArgs{TokenStorage: AWSTokenStorage})

		assert.Nil(t, ss)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "AWS")
	})

}
