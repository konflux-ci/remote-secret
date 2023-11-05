/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	esoframework "github.com/external-secrets/external-secrets-e2e/framework"
	. "github.com/onsi/ginkgo/v2"
)

// Compose helps define multiple testcases with same/different auth methods.
func Compose(descAppend string, f *esoframework.Framework, fn func(f *esoframework.Framework) (string, func(*RsTestCase)), tweaks ...func(*RsTestCase)) TableEntry {
	// prepend common fn to tweaks
	desc, cfn := fn(f)
	tweaks = append([]func(*RsTestCase){cfn}, tweaks...)

	// need to convert []func to []interface{}
	ifs := make([]interface{}, len(tweaks))
	for i := 0; i < len(tweaks); i++ {
		ifs[i] = tweaks[i]
	}
	te := Entry(desc+" "+descAppend, ifs...)

	return te
}
