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

import "strings"

// NameCorresponds is a simple helper function to figure out whether the provided
// `actualName` can be a name of an K8s object with the provided `specificName` (`metadata.name`)
// or `generateName` (`metadata.generateName`).
//
// The equality of the actualName with the specificName is determined first and only then
// the generateName is considered. This is to conform with the behavior of the cluster
// (https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/object-meta/).
func NameCorresponds(actualName, specificName, generateName string) bool {
	if specificName != "" {
		return actualName == specificName
	}

	if generateName != "" {
		return strings.HasPrefix(actualName, generateName)
	}

	// both specific name and generate name are empty. This means that the actualName just is
	// what it is and so we can only say that it corresponds to itself.
	return true
}
