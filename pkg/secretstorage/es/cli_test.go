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

package es

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalSucceeds(t *testing.T) {

	jsonString := `{"fake":{"data":[{"key":"key", "value":"val1", "valueMap":{"k1":"v1"} }]}}`

	res, err := NewESSecretStorage(context.TODO(), nil, "", jsonString)

	assert.NotNil(t, res)
	assert.Nil(t, err)
}

func TestUnmarshalFailed(t *testing.T) {

	jsonString := `{"not_existed":{}}}`

	res, err := NewESSecretStorage(context.TODO(), nil, "", jsonString)

	assert.Nil(t, res)
	assert.NotNil(t, err)
}
