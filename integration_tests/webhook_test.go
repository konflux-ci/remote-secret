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

package integrationtests

//tests for validating and mutating webhooks
import (
	"encoding/base64"

	"github.com/metlos/crenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MutatorTest", func() {

	Describe("Upload token", func() {
		When("RemoteSecret with upload data fields", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{},
				MonitoredObjectTypes: []client.Object{
					&api.RemoteSecret{},
				},
			}

			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should store data on remote secret creation", func() {

				rs := &api.RemoteSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-remote-secret",
						Namespace: "default",
					},
					UploadData: map[string]string{
						"test":  base64.StdEncoding.EncodeToString([]byte("test1")),
						"test2": base64.StdEncoding.EncodeToString([]byte("test1")),
					},
				}
				Expect(ITest.Client.Create(ITest.Context, rs)).To(Succeed())
				data, err := ITest.Storage.Get(ITest.Context, rs)
				Expect(err).NotTo(HaveOccurred())
				Expect(*data).To(HaveLen(2))
			})
		})
	})
})
