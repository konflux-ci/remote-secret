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

import (
	"github.com/metlos/crenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("TokenUploadController", func() {
	Describe("Upload token", func() {

		When("no secret exists", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&v1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret-upload",
							Namespace: "default",
							Labels: map[string]string{"appstudio.redhat.com/remotesecret-name": "new-remote-secret",
								"appstudio.redhat.com/remotesecret-target-namespace": "mshaposh-tenant"},
						},
						Type: "Opaque",
						Data: map[string][]byte{"a": []byte("b")},
					},
				},
			}

			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("creates new one", func() {
				Eventually(func(g Gomega) {
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)).To(HaveLen(1))
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)[0].Name).To(Equal("new-remote-secret"))
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)[0].Spec.Targets[0].Namespace).To(Equal("mshaposh-tenant"))
				})
			})
		})

		When("secret is exists", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "new-remote-secret",
							Namespace: "default",
						},
					},
				},
			}

			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})
			
			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("adds new target", func() {
				o := &v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-remote-secret-upload",
						Namespace: "default",
						Labels: map[string]string{"appstudio.redhat.com/remotesecret-name": "new-remote-secret",
							"appstudio.redhat.com/remotesecret-target-namespace": "mshaposh-tenant"},
					},
					Type: "Opaque",
					Data: map[string][]byte{"a": []byte("b")},
				}

				Expect(ITest.Client.Create(ITest.Context, o)).To(Succeed())
				Eventually(func(g Gomega) {
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)).To(HaveLen(1))
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)[0].Spec.Targets[0].Namespace).To(Equal("mshaposh-tenant"))
				})
			})
		})

	})

})
