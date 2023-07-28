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
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("RemoteSecret", func() {
	Describe("Create", func() {
		When("no targets", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
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

			It("succeeds", func() {
				Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)).To(HaveLen(1))
			})
		})

		When("with targets", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Targets: []api.RemoteSecretTarget{
								{
									Namespace: "ns",
								},
							},
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

			It("succeeds", func() {
				Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)).To(HaveLen(1))
			})
		})

		When("secret data uploaded", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
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

			It("should have secret data keys in status", func() {
				// Store secret data
				data := &remotesecretstorage.SecretData{
					"a": []byte("b"),
				}
				Expect(ITest.Storage.Store(ITest.Context, *crenv.First[*api.RemoteSecret](&test.InCluster), data)).To(Succeed())

				// Check that status contains the key from secret data
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					Expect(rs).NotTo(BeNil())
					Expect(rs.Status.SecretStatus.Keys).To(HaveLen(1))
					Expect(rs.Status.SecretStatus.Keys[0]).To(Equal("a"))
				})
			})
		})
	})

	Describe("Update", func() {
		When("target removed", func() {
			var test crenv.TestSetup
			var targetA, targetB string

			BeforeEach(func() {
				targetA = string(uuid.NewUUID())
				targetB = string(uuid.NewUUID())
				Expect(ITest.Client.Create(ITest.Context, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: targetA},
				})).To(Succeed())
				Expect(ITest.Client.Create(ITest.Context, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: targetB},
				})).To(Succeed())

				test = crenv.TestSetup{
					ToCreate: []client.Object{
						&api.RemoteSecret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-remote-secret",
								Namespace: "default",
							},
							Spec: api.RemoteSecretSpec{
								Secret: api.LinkableSecretSpec{
									Name: "injected-secret",
									LinkedTo: []api.SecretLink{{
										ServiceAccount: api.ServiceAccountLink{
											Managed: api.ManagedServiceAccountSpec{
												Name: "injected-sa",
											},
										}}},
								},
								Targets: []api.RemoteSecretTarget{{
									Namespace: targetA,
								}, {
									Namespace: targetB,
								}},
							},
						},
					},
				}

				test.BeforeEach(ITest.Context, ITest.Client, nil)
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				Expect(rs).NotTo(BeNil())
				Expect(ITest.Storage.Store(ITest.Context, rs, &remotesecretstorage.SecretData{
					"a": []byte("b"),
				})).To(Succeed())
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should remove the target secret", func() {
				// check that secret is created in each target
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetA}, &corev1.Secret{})).To(Succeed())
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetB}, &corev1.Secret{})).To(Succeed())
				})

				// now remove targetB from the spec
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				Expect(rs).NotTo(BeNil())
				rs.Spec.Targets = []api.RemoteSecretTarget{{Namespace: targetA}}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs.Status.Targets).To(HaveLen(1))
					// check that the secret in targetA is still there but not in targetB
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetA}, &corev1.Secret{})).To(Succeed())
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetB}, &corev1.Secret{})).Error()
				})
			})

			It("should remove the managed service account", func() {
				// check that service account is created in each target
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-sa", Namespace: targetA}, &corev1.ServiceAccount{})).To(Succeed())
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-sa", Namespace: targetB}, &corev1.ServiceAccount{})).To(Succeed())
				})

				// now remove targetB from the spec
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				Expect(rs).NotTo(BeNil())
				rs.Spec.Targets = []api.RemoteSecretTarget{{Namespace: targetA}}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// check that the service account in targetA is still there but not in targetB
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs.Status.Targets).To(HaveLen(1))
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-sa", Namespace: targetA}, &corev1.ServiceAccount{})).To(Succeed())
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-sa", Namespace: targetB}, &corev1.ServiceAccount{})).Error()
				})
			})
		})

		When("target added", func() {
		})

		When("secret spec changed", func() {
		})

		When("linked SAs changed", func() {
		})
	})

	Describe("Delete", func() {
		When("no targets present", func() {
		})

		When("targets present", func() {
		})
	})
})
