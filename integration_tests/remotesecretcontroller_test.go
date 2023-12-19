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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var remoteSecretReconciliationTrigger = map[schema.GroupKind]func(client.Object){
	{Group: api.GroupVersion.Group, Kind: "RemoteSecret"}: func(o client.Object) {
		rs := o.(*api.RemoteSecret)
		if rs.Spec.Secret.Annotations == nil {
			rs.Spec.Secret.Annotations = map[string]string{}
		}
		rs.Spec.Secret.Annotations["reconcile-trigger"] = string(uuid.NewUUID())
	},
}

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
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
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
					ReconciliationTrigger: remoteSecretReconciliationTrigger,
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
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetB}, &corev1.Secret{})).NotTo(Succeed())
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
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-sa", Namespace: targetB}, &corev1.ServiceAccount{})).NotTo(Succeed())
				})
			})

			It("should remove all targets secrets", func() {
				// remove all targets secrets from the spec
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				Expect(rs).NotTo(BeNil())
				rs.Spec.Targets = []api.RemoteSecretTarget{}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs.Status.Targets).To(BeEmpty())
					// check that rs.Status.Conditions Deployed is False
					g.Expect(meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDeployed))).To(BeFalse())
					// check that all secrets are removed
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetA}, &corev1.Secret{})).NotTo(Succeed())
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKey{Name: "injected-secret", Namespace: targetB}, &corev1.Secret{})).NotTo(Succeed())
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
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				// first change the spec to contain labels
				rs.Spec.Secret.Labels = map[string]string{
					"k1": "v1",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// and test that the secret got updated with the label
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
						g.Expect(secrets[0].Labels).NotTo(BeNil())
						g.Expect(secrets[0].Labels).To(HaveKeyWithValue("k1", "v1"))
					}
				})

				// now override the labels in the target to contain no labels
				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					Labels: &map[string]string{},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// and check that the labels are no longer present on the secret
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
						// the secret will have a label marking it as linked to the remote secret
						// but it should not contain the labels defined by the secret spec of the RS.
						g.Expect(secrets[0].Labels).NotTo(HaveKey("k1"))
					}
				})
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
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)

				// first change the spec to contain annotations
				rs.Spec.Secret.Annotations = map[string]string{
					"k1": "v1",
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// and test that the secret got updated with the annotation
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
						g.Expect(secrets[0].Annotations).NotTo(BeNil())
						g.Expect(secrets[0].Annotations).To(HaveKeyWithValue("k1", "v1"))
					}
				})

				// now override the labels in the target to not contain annotations
				rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				rs.Spec.Targets[0].Secret = &api.SecretOverride{
					Annotations: &map[string]string{},
				}
				Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())

				// and check that the annotations are no longer present on the secret
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					if len(secrets) > 0 {
						g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
						g.Expect(secrets[0].Annotations).NotTo(HaveKey("k1"))
					}
				})
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
				Skip("not supported yet")
				// rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				//
				// rs.Spec.Targets[0].Secret = &api.SecretOverride{
				// 	LinkedTo: &[]api.SecretLink{
				// 		{
				// 			ServiceAccount: api.ServiceAccountLink{
				// 				Managed: api.ManagedServiceAccountSpec{
				// 					GenerateName: "managed-sa-",
				// 				},
				// 			},
				// 		},
				// 	},
				// }
				// Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())
				//
				// test.SettleWithCluster(ITest.Context, func(g Gomega) {
				// 	secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
				// 	g.Expect(secrets).To(HaveLen(1))
				// 	if len(secrets) > 0 {
				// 		g.Expect(secrets[0].Name).To(HavePrefix("spec-secret-"))
				// 	}
				// 	sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
				// 	g.Expect(sas).To(HaveLen(1))
				// 	if len(sas) > 0 {
				// 		g.Expect(sas[0].Name).To(HavePrefix("managed-sa-"))
				// 	}
				// })
			})
			It("can remove the linked SAs", func() {
				Skip("not supported yet")
				// rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				//
				// rs.Spec.Targets[0].Secret = &api.SecretOverride{
				// 	LinkedTo: &[]api.SecretLink{
				// 		{
				// 			ServiceAccount: api.ServiceAccountLink{
				// 				Managed: api.ManagedServiceAccountSpec{
				// 					GenerateName: "managed-sa-",
				// 				},
				// 			},
				// 		},
				// 	},
				// }
				// Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())
				//
				// test.SettleWithCluster(ITest.Context, func(g Gomega) {
				// 	secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
				// 	g.Expect(secrets).To(HaveLen(1))
				// 	sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
				// 	g.Expect(sas).To(HaveLen(1))
				// })
				//
				// // now set the LinkedTo to an empty array
				// rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				// rs.Spec.Targets[0].Secret.LinkedTo = nil
				// Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())
				//
				// test.SettleWithCluster(ITest.Context, func(g Gomega) {
				// 	sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
				// 	g.Expect(sas).To(BeEmpty())
				// })
			})
			It("can override the linked SAs to empty", func() {
				Skip("not supported yet")
				// 	rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				//
				// 	// first update the secret spec to have a SA link
				// 	rs.Spec.Secret.LinkedTo = []api.SecretLink{
				// 		{
				// 			ServiceAccount: api.ServiceAccountLink{
				// 				Managed: api.ManagedServiceAccountSpec{
				// 					GenerateName: "managed-sa-",
				// 				},
				// 			},
				// 		},
				// 	}
				// 	Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())
				// 	test.SettleWithCluster(ITest.Context, func(g Gomega) {
				// 		sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
				// 		g.Expect(sas).To(HaveLen(1))
				// 	})
				//
				// 	// now, try to remove the link by specifying an empty link array at the target
				// 	// while leaving the link in the secret spec of the remote secret.
				// 	rs = *crenv.First[*api.RemoteSecret](&test.InCluster)
				// 	rs.Spec.Targets[0].Secret = &api.SecretOverride{
				// 		LinkedTo: &[]api.SecretLink{},
				// 	}
				// 	Expect(ITest.Client.Update(ITest.Context, rs)).To(Succeed())
				//
				// 	test.SettleWithCluster(ITest.Context, func(g Gomega) {
				// 		sas := crenv.GetAll[*corev1.ServiceAccount](&test.InCluster)
				// 		g.Expect(sas).To(BeEmpty())
				// 	})
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
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
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
					g.Expect(rs.Status.Conditions).To(HaveLen(2))
					g.Expect(meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDataObtained))).To(BeTrue())
				})
			})

			It("should report in condition if data is missing in storage", func() {
				// Check that data obtained
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(rs.Status.Conditions).To(HaveLen(2))
					g.Expect(meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDataObtained))).To(BeTrue())
				})

				Expect(ITest.Storage.Delete(ITest.Context, *crenv.First[*api.RemoteSecret](&test.InCluster))).To(Succeed())

				// Check that data is missing
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(rs.Status.Conditions).To(HaveLen(2))
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
	Describe("Interactions", func() {
		When("two RemoteSecrets have the same target", func() {
			originalRS := defaultRemoteSecretWithTarget("original-rs", "secret-abc", "default")
			test := crenv.TestSetup{
				ToCreate:              []client.Object{originalRS},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("second RemoteSecret should have error in target", func() {
				duplicateRS := defaultRemoteSecretWithTarget("duplicate-target-rs", "secret-abc", "default")
				Expect(ITest.Client.Create(ITest.Context, duplicateRS)).To(Succeed())

				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					remoteSecrets := crenv.FindByNamePrefix[*api.RemoteSecret](&test.InCluster, client.ObjectKeyFromObject(originalRS))
					g.Expect(remoteSecrets).To(HaveLen(1))
					Expect(remoteSecrets[0].Status.Targets[0].Error).To(BeEmpty())
					Expect(remoteSecrets[0].Status.Targets[0].DeployedSecret.Name).To(Equal(originalRS.Spec.Secret.Name))

					remoteSecrets = crenv.FindByNamePrefix[*api.RemoteSecret](&test.InCluster, client.ObjectKeyFromObject(duplicateRS))
					g.Expect(remoteSecrets).To(HaveLen(1))
					Expect(remoteSecrets[0].Status.Targets[0].Error).NotTo(BeEmpty())
				})
			})
		})
	})

	Describe("Conditions", func() {
		When("data obtained but no targets present", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
					},
				},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
				uploadArbitraryDataToRS(&test)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should report true Deployed type condition with NoTargets reason", func() {
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					cond := meta.FindStatusCondition(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDeployed))
					g.Expect(cond).NotTo(BeNil())
					g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
					g.Expect(cond.Reason).To(Equal(string(api.RemoteSecretReasonNoTargets)))
				})
			})
		})

		When("one of the targets fails to deploy", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								GenerateName: "secret-from-rs-",
							},
							Targets: []api.RemoteSecretTarget{{
								Namespace: "default",
							}, {
								Namespace: "other", // namespace does not exist, will fail to deploy
							}},
						},
					},
				},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
				uploadArbitraryDataToRS(&test)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should report false Deployed type condition with PartiallyInjected reason", func() {
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					cond := meta.FindStatusCondition(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDeployed))
					g.Expect(cond).NotTo(BeNil())
					g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
					g.Expect(cond.Reason).To(Equal(string(api.RemoteSecretReasonPartiallyInjected)))
				})
			})
		})

		When("only present target fails to deploy", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								GenerateName: "secret-from-rs-",
							},
							Targets: []api.RemoteSecretTarget{{
								Namespace: "other", // namespace does not exist, will fail to deploy
							}},
						},
					},
				},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
				uploadArbitraryDataToRS(&test)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should report false Deployed type condition with Error reason", func() {
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					cond := meta.FindStatusCondition(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDeployed))
					g.Expect(cond).NotTo(BeNil())
					g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
					g.Expect(cond.Reason).To(Equal(string(api.RemoteSecretReasonError)))
				})
			})
		})
	})

	Describe("ExpectedSecret in target Status", func() {
		When("secret is successfully deployed to target", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								Name:         "exact-secret-name",
								GenerateName: "not-used-generate-",
							},
							Targets: []api.RemoteSecretTarget{{
								Namespace: "default",
							}},
						},
					},
				},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
				uploadArbitraryDataToRS(&test)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("shows just the actual SecretName, without ExpectedSecret", func() {
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(rs.Status.Targets).To(HaveLen(1))
					g.Expect(rs.Status.Targets[0].SecretName).To(Equal("exact-secret-name")) //nolint:staticcheck // SA1019 - this deprecated field needs to be set
					g.Expect(rs.Status.Targets[0].ExpectedSecret).To(BeNil())
				})
			})
		})

		When("target with secret override fails to deploy", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								Name:         "expected-name",
								GenerateName: "expected-generate-",
							},
							Targets: []api.RemoteSecretTarget{{
								Namespace: "non-existing",
							}},
						},
					},
				},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
				uploadArbitraryDataToRS(&test)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should should show ExpectedSecret name for the failed target", func() {
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(rs.Status.Targets).To(HaveLen(1))
					g.Expect(rs.Status.Targets[0].SecretName).To(Equal("")) //nolint:staticcheck // SA1019 - this deprecated field needs to be set
					g.Expect(rs.Status.Targets[0].ExpectedSecret.Name).To(Equal("expected-name"))
					g.Expect(rs.Status.Targets[0].ExpectedSecret.GenerateName).To(Equal("expected-generate-"))
				})
			})
		})

		When("target with secret override fails to deploy", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								Name:         "not-used-name",
								GenerateName: "not-used-generate-",
							},
							Targets: []api.RemoteSecretTarget{
								{
									Namespace: "non-existing",
									Secret: &api.SecretOverride{
										Name:         "expected-override",
										GenerateName: "expected-generate-",
									},
								},
							},
						},
					},
				},
				ReconciliationTrigger: remoteSecretReconciliationTrigger,
			}
			BeforeEach(func() {
				test.BeforeEach(ITest.Context, ITest.Client, nil)
				uploadArbitraryDataToRS(&test)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("should should show ExpectedSecret name for the failed target", func() {
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
					g.Expect(rs).NotTo(BeNil())
					g.Expect(rs.Status.Targets).To(HaveLen(1))
					g.Expect(rs.Status.Targets[0].SecretName).To(Equal("")) //nolint:staticcheck // SA1019 - this deprecated field needs to be set
					g.Expect(rs.Status.Targets[0].ExpectedSecret.Name).To(Equal("expected-override"))
					g.Expect(rs.Status.Targets[0].ExpectedSecret.GenerateName).To(Equal("expected-generate-"))
				})
			})
		})
	})
})

func uploadArbitraryDataToRS(test *crenv.TestSetup) {
	rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
	Expect(rs).NotTo(BeNil())
	Expect(ITest.Storage.Store(ITest.Context, rs, &remotesecretstorage.SecretData{
		"a": []byte("b"),
	})).To(Succeed())
}

func defaultRemoteSecretWithTarget(rsName, secretName, targetNs string) *api.RemoteSecret {
	return &api.RemoteSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rsName,
			Namespace: "default",
		},
		Spec: api.RemoteSecretSpec{
			Secret: api.LinkableSecretSpec{
				Name: secretName,
			},
			Targets: []api.RemoteSecretTarget{
				{
					Namespace: targetNs,
				},
			},
		},
		UploadData: map[string][]byte{
			"k1": []byte("v1"),
		},
	}
}
