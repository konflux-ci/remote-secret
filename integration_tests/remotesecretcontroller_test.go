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
	"k8s.io/apimachinery/pkg/api/meta"
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

		When("with target overrides", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								Name: "spec-name",
							},
							Targets: []api.RemoteSecretTarget{
								{
									Namespace: "default",
									Secret: &api.SecretOverride{
										Name: "target-name",
									},
								},
							},
						},
						UploadData: map[string][]byte{
							"k1": []byte("v1"),
							"k2": []byte("v2"),
						},
					},
				},
				MonitoredObjectTypes: []client.Object{
					&corev1.Secret{},
				},
			}

			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("succeeds", func() {
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secPtr := crenv.First[*corev1.Secret](&test.InCluster)
					g.Expect(secPtr).NotTo(BeNil())
					sec := *secPtr
					g.Expect(sec.Name).To(Equal("target-name"))
					g.Expect(sec.Data).To(HaveLen(2))
					g.Expect(sec.Data["k1"]).To(Equal([]byte("v1")))
					g.Expect(sec.Data["k2"]).To(Equal([]byte("v2")))
				})
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
					g.Expect(rs).NotTo(BeNil())
					g.Expect(rs.Status.SecretStatus.Keys).To(HaveLen(1))
					g.Expect(rs.Status.SecretStatus.Keys[0]).To(Equal("a"))
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
										},
									}},
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

				// check that secret is created in each target
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetA}, &corev1.Secret{})).To(Succeed())
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetB}, &corev1.Secret{})).To(Succeed())
				})
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should remove the target secret", func() {
				// remove targetB from the spec
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
				// remove targetB from the spec
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

		When("overriden targets", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "rs",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								GenerateName: "spec-secret-",
							},
							Targets: []api.RemoteSecretTarget{
								{
									Namespace: "default",
								},
							},
						},
						UploadData: map[string][]byte{
							"k1": []byte("v1"),
							"k2": []byte("v2"),
						},
					},
				},
				MonitoredObjectTypes: []client.Object{
					&corev1.Secret{}, &corev1.ServiceAccount{},
				},
			}

			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("changes overriden target to non-overriden", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				// first change the target to have an override
				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					GenerateName: "target-secret-",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("target-secret-"))
					}
				})

				// and now remove the override
				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret = nil
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
					}
				})
			})
			It("updates the labels", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				// first change the target to have an override
				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					Labels: &map[string]string{
						"k1": "v1",
					},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// and test that the secret got updated with the label
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
						g.Expect(secrets[0].Labels).NotTo(BeNil())
						g.Expect(secrets[0].Labels["k1"]).To(Equal("v1"))
					}
				})
			})
			It("can remove the labels", func() {
				Skip("not supported atm")
			})
			It("updates the annotations", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				// first change the target to have an override
				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					Annotations: &map[string]string{
						"k1": "v1",
					},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// and test that the secret got updated with the label
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
						g.Expect(secrets[0].Annotations).NotTo(BeNil())
						g.Expect(secrets[0].Annotations["k1"]).To(Equal("v1"))
					}
				})
			})
			It("can remove the annotations", func() {
				Skip("not supported atm")
			})
			It("override the name", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					Name: "target-secret",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(Equal("target-secret"))
					}
				})
			})
			It("update the name override", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					Name: "target-secret",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(Equal("target-secret"))
					}
				})

				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret.Name = "changed-target-secret"
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(Equal("changed-target-secret"))
					}
				})
			})
			It("override the generateName", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					GenerateName: "target-secret-",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("target-secret-"))
					}
				})
			})
			It("update the generateName override", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					GenerateName: "target-secret-",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("target-secret-"))
					}
				})

				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret.GenerateName = "changed-target-secret-"
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("changed-target-secret-"))
					}
				})
			})
			It("updates the linked SAs", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					LinkedTo: &[]api.SecretLink{
						{
							ServiceAccount: api.ServiceAccountLink{
								Managed: api.ManagedServiceAccountSpec{
									GenerateName: "managed-sa-",
								},
							},
						},
					},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
					}
					sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
					g.Expect(sas).To(HaveLen(1))
					if len(sas) > 0 {
						g.Expect(sas[0].Name).To(HavePrefix("managed-sa-"))
					}
				})
			})
			It("can remove the linked SAs", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					LinkedTo: &[]api.SecretLink{
						{
							ServiceAccount: api.ServiceAccountLink{
								Managed: api.ManagedServiceAccountSpec{
									GenerateName: "managed-sa-",
								},
							},
						},
					},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
					g.Expect(sas).To(HaveLen(1))
				})

				// now set the LinkedTo to an empty array
				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret.LinkedTo = nil
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
					g.Expect(sas).To(BeEmpty())
				})
			})
			It("can override the linked SAs to empty", func() {
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				// first update the secret spec to have a SA link
				rs.Spec.Secret.LinkedTo = []api.SecretLink{
					{
						ServiceAccount: api.ServiceAccountLink{
							Managed: api.ManagedServiceAccountSpec{
								GenerateName: "managed-sa-",
							},
						},
					},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
					g.Expect(sas).To(HaveLen(1))
				})

				// now, try to remove the link by specifying an empty link array at the target
				// while leaving the link in the secret spec of the remote secret.
				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					LinkedTo: &[]api.SecretLink{},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
					g.Expect(sas).To(BeEmpty())
				})
			})
		})
	})

	Describe("READ", func() {
		When("data in storage", func() {
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
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				Expect(rs).NotTo(BeNil())
				Expect(ITest.Storage.Store(ITest.Context, rs, &remotesecretstorage.SecretData{
					"a": []byte("b"),
				})).To(Succeed())
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})
			It("should report in condition if data is in storage", func() {
				// Check that data obtained
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(len(rs.Status.Conditions)).To(Equal(2))
					g.Expect(meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDataObtained))).To(BeTrue())
				})
			})

			It("should report in condition if data is missing in storage", func() {
				// Check that data obtained
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(len(rs.Status.Conditions)).To(Equal(2))
					g.Expect(meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDataObtained))).To(BeTrue())
				})

				Expect(ITest.Storage.Delete(ITest.Context, *crenv.First[*api.RemoteSecret](&test.InCluster))).To(Succeed())

				// Check that data is missing
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(len(rs.Status.Conditions)).To(Equal(2))
					g.Expect(meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDataObtained))).To(BeFalse())
				})
			})
		})
	})
	Describe("Delete", func() {
		When("no targets present", func() {
		})

		When("targets present", func() {
		})
	})
})
