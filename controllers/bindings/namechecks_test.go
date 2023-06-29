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

package bindings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameCorresponds(t *testing.T) {
	t.Run("name check has precedence", func(t *testing.T) {
		// Kubernetes docs (https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/) specify that generateName is only taken into account if the name is not
		// specified. We need to behave the same.
		assert.True(t, NameCorresponds("a", "a", "asdf"))
		assert.False(t, NameCorresponds("a-suffix", "b", "a-"))
	})

	t.Run("name check with empty generateName", func(t *testing.T) {
		assert.True(t, NameCorresponds("a", "a", ""))
		assert.False(t, NameCorresponds("a", "b", ""))
	})

	t.Run("uses generate if name empty", func(t *testing.T) {
		assert.True(t, NameCorresponds("a", "", "a"))
		assert.True(t, NameCorresponds("afbsfdf", "", "a"))
		assert.False(t, NameCorresponds("a", "", "b"))
		assert.False(t, NameCorresponds("abbasdf", "", "b"))
	})

	t.Run("corresponds if no requirements given", func(t *testing.T) {
		assert.True(t, NameCorresponds("a", "", ""))
	})
}
