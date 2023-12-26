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

var _ = Describe("TokenUploadController", func() {
	Describe("Upload token", func() {
		When("no RemoteSecret exists", func() {
			test := crenv.TestSetup{
				ToCreate: []client.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret-upload",
							Namespace: "default",
							Labels:    map[string]string{api.UploadSecretLabel: "remotesecret"},
							Annotations: map[string]string{
								api.RemoteSecretNameAnnotation: "new-remote-secret",
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{"a": []byte("b")},
					},
				},
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

			It("creates a new RemoteSecret", func() {
				// we are waiting cluster to settle down. At this moment Secret controller is ready to create RemoteSecret
				// in method createRemoteSecret. But not yet creating it. So we need to wait for it.
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)).To(HaveLen(1))
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)[0].Name).To(Equal("new-remote-secret"))
				})
			})

		})
		When("secret data are already present", func() {
			var test crenv.TestSetup
			var target string
			oldSecretData := map[string][]byte{
				"a": []byte("b"),
			}
			newSecretData := map[string][]byte{
				"x": []byte("foo"),
			}
			sd := remotesecretstorage.SecretData(oldSecretData)
			uploadSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-remote-secret-upload",
					Namespace:   "default",
					Labels:      map[string]string{api.UploadSecretLabel: "remotesecret"},
					Annotations: map[string]string{api.RemoteSecretNameAnnotation: "test-remote-secret"},
				},
				Data: newSecretData,
			}

			BeforeEach(func() {
				target = string(uuid.NewUUID())
				Expect(ITest.Client.Create(ITest.Context, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: target},
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
								},
								Targets: []api.RemoteSecretTarget{{
									Namespace: target,
								}},
							},
						},
					},
					MonitoredObjectTypes: []client.Object{
						&corev1.Secret{},
					},
					ReconciliationTrigger: remoteSecretReconciliationTrigger,
				}

				test.BeforeEach(ITest.Context, ITest.Client, nil)
				rs := *crenv.First[*api.RemoteSecret](&test.InCluster)
				Expect(rs).NotTo(BeNil())
				Expect(ITest.Storage.Store(ITest.Context, rs, &sd)).To(Succeed())
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			It("updates the target secret with new data", func() {
				// check that target secret contains the old data
				test.ReconcileWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					g.Expect(secrets[0].Data).To(Equal(oldSecretData))
				})

				// create upload secret
				Expect(ITest.Client.Create(ITest.Context, uploadSecret)).To(Succeed())

				// check that after uploading, the data in secret are updated
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					secrets := crenv.GetAll[*corev1.Secret](&test.InCluster)
					g.Expect(secrets).To(HaveLen(1))
					g.Expect(secrets[0].Data).To(Equal(newSecretData))
				})
			})
		})
		When("upload secret is invalid ", func() {
			test := crenv.TestSetup{
				MonitoredObjectTypes: []client.Object{&corev1.Secret{}, &corev1.Event{}},
			}
			// Define the secret here to avoid repetition and only overwrite the specific parts in each test case.
			uploadSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-remote-secret-upload",
					Namespace: "default",
					Labels: map[string]string{
						"appstudio.redhat.com/upload-secret": "remotesecret",
					},
					Annotations: map[string]string{
						"appstudio.redhat.com/remotesecret-name": "test-remote-secret",
					},
				},
			}

			expectUploadFailed := func() {
				test.SettleWithCluster(ITest.Context, func(g Gomega) {
					// Upload secret should be deleted
					g.Expect(crenv.GetAll[*corev1.Secret](&test.InCluster)).To(BeEmpty())
					// There is only one RS present in the cluster (no new one created)
					g.Expect(crenv.GetAll[*api.RemoteSecret](&test.InCluster)).To(HaveLen(1))

					// RS is still in awaiting data state
					g.Expect((*crenv.First[*api.RemoteSecret](&test.InCluster)).Status.Conditions).To(HaveLen(1))
					g.Expect((*crenv.First[*api.RemoteSecret](&test.InCluster)).Status.Conditions[0].Reason).To(Equal(string(api.RemoteSecretReasonAwaitingTokenData)))

					// Error event should be created
					event := &corev1.Event{}
					g.Expect(ITest.Client.Get(ITest.Context, client.ObjectKeyFromObject(uploadSecret), event)).To(Succeed())
					g.Expect(event.Type).To(Equal("Error"))
				})
			}

			BeforeEach(func() {
				test.ToCreate = []client.Object{ // new RS for each test
					&api.RemoteSecret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-remote-secret",
							Namespace: "default",
						},
						Spec: api.RemoteSecretSpec{
							Secret: api.LinkableSecretSpec{
								Type:         corev1.SecretTypeSSHAuth,
								RequiredKeys: []api.SecretKey{{Name: "foo"}},
							},
						},
					},
				}
				test.BeforeEach(ITest.Context, ITest.Client, nil)
			})

			AfterEach(func() {
				test.AfterEach(ITest.Context)
			})

			When("secret types do not match", func() {
				uploadSecret := uploadSecret.DeepCopy()
				uploadSecret.Type = corev1.SecretTypeOpaque
				uploadSecret.Data = map[string][]byte{corev1.SSHAuthPrivateKey: []byte("ssh..."), "foo": []byte("bar")}

				It("fails the upload", func() {
					Expect(ITest.Client.Create(ITest.Context, uploadSecret)).To(Succeed())
					expectUploadFailed()
				})
			})

			When("upload secret does not have required key foo", func() {
				uploadSecret := uploadSecret.DeepCopy()
				uploadSecret.Type = corev1.SecretTypeSSHAuth
				uploadSecret.Data = map[string][]byte{corev1.SSHAuthPrivateKey: []byte("ssh...")}

				It("fails the upload", func() {
					Expect(ITest.Client.Create(ITest.Context, uploadSecret)).To(Succeed())
					expectUploadFailed()
				})
			})
		})
	})

	Describe("Partial Update", func() {
		testSetup := crenv.TestSetup{
			ToCreate: []client.Object{
				&api.RemoteSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rs",
						Namespace: "default",
					},
				},
			},
			MonitoredObjectTypes: []client.Object{&corev1.Secret{}},
		}

		BeforeEach(func() {
			testSetup.BeforeEach(ITest.Context, ITest.Client, nil)

			Expect(ITest.Client.Create(ITest.Context, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "default",
					Labels: map[string]string{
						api.UploadSecretLabel: "remotesecret",
					},
					Annotations: map[string]string{
						api.RemoteSecretNameAnnotation: "rs",
					},
				},
				Data: map[string][]byte{
					"a": []byte("va"),
					"b": []byte("vb"),
				},
			})).To(Succeed())

			testSetup.SettleWithCluster(ITest.Context, func(g Gomega) {
				rs := *crenv.First[*api.RemoteSecret](&testSetup.InCluster)
				cond := meta.FindStatusCondition(rs.Status.Conditions, "DataObtained")

				g.Expect(cond).NotTo(BeNil())
				g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			})
		})

		AfterEach(func() {
			testSetup.AfterEach(ITest.Context)
		})

		It("creates new data entries", func() {
			Expect(ITest.Client.Create(ITest.Context, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "default",
					Labels: map[string]string{
						api.UploadSecretLabel: "remotesecret",
					},
					Annotations: map[string]string{
						api.RemoteSecretNameAnnotation:          "rs",
						api.RemoteSecretPartialUpdateAnnotation: "true",
					},
				},
				Data: map[string][]byte{
					"c": []byte("vc"),
				},
			})).To(Succeed())

			testSetup.SettleWithCluster(ITest.Context, func(g Gomega) {
				rs := *crenv.First[*api.RemoteSecret](&testSetup.InCluster)
				data, err := ITest.Storage.Get(ITest.Context, rs)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(*data).To(HaveLen(3))
				g.Expect(*data).To(HaveKey("a"))
				g.Expect(*data).To(HaveKey("b"))
				g.Expect(*data).To(HaveKey("c"))
				g.Expect((*data)["a"]).To(Equal([]byte("va")))
				g.Expect((*data)["b"]).To(Equal([]byte("vb")))
				g.Expect((*data)["c"]).To(Equal([]byte("vc")))
			})
		})

		It("updates existing entries", func() {
			Expect(ITest.Client.Create(ITest.Context, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "default",
					Labels: map[string]string{
						api.UploadSecretLabel: "remotesecret",
					},
					Annotations: map[string]string{
						api.RemoteSecretNameAnnotation:          "rs",
						api.RemoteSecretPartialUpdateAnnotation: "true",
					},
				},
				Data: map[string][]byte{
					"b": []byte("vb_new"),
				},
			})).To(Succeed())

			testSetup.SettleWithCluster(ITest.Context, func(g Gomega) {
				rs := *crenv.First[*api.RemoteSecret](&testSetup.InCluster)
				data, err := ITest.Storage.Get(ITest.Context, rs)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(*data).To(HaveLen(2))
				g.Expect(*data).To(HaveKey("a"))
				g.Expect(*data).To(HaveKey("b"))
				g.Expect((*data)["a"]).To(Equal([]byte("va")))
				g.Expect((*data)["b"]).To(Equal([]byte("vb_new")))
			})
		})

		It("deletes entries", func() {
			Expect(ITest.Client.Create(ITest.Context, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "default",
					Labels: map[string]string{
						api.UploadSecretLabel: "remotesecret",
					},
					Annotations: map[string]string{
						api.RemoteSecretNameAnnotation:          "rs",
						api.RemoteSecretPartialUpdateAnnotation: "true",
						api.RemoteSecretDeletedKeysAnnotation:   "a,b",
					},
				},
			})).To(Succeed())

			testSetup.SettleWithCluster(ITest.Context, func(g Gomega) {
				rs := *crenv.First[*api.RemoteSecret](&testSetup.InCluster)
				data, err := ITest.Storage.Get(ITest.Context, rs)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(*data).To(BeEmpty())
			})
		})

		It("create update delete in one go", func() {
			Expect(ITest.Client.Create(ITest.Context, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "default",
					Labels: map[string]string{
						api.UploadSecretLabel: "remotesecret",
					},
					Annotations: map[string]string{
						api.RemoteSecretNameAnnotation:          "rs",
						api.RemoteSecretPartialUpdateAnnotation: "true",
						api.RemoteSecretDeletedKeysAnnotation:   "a",
					},
				},
				Data: map[string][]byte{
					"b": []byte("vb_new"),
					"c": []byte("vc"),
				},
			})).To(Succeed())

			testSetup.SettleWithCluster(ITest.Context, func(g Gomega) {
				rs := *crenv.First[*api.RemoteSecret](&testSetup.InCluster)
				data, err := ITest.Storage.Get(ITest.Context, rs)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(*data).To(HaveLen(2))
				g.Expect(*data).To(HaveKey("b"))
				g.Expect(*data).To(HaveKey("c"))
				g.Expect((*data)["b"]).To(Equal([]byte("vb_new")))
				g.Expect((*data)["c"]).To(Equal([]byte("vc")))
			})
		})
		It("create update delete with different secret type than remote secret", func() {
			Expect(ITest.Client.Create(ITest.Context, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sec",
					Namespace: "default",
					Labels: map[string]string{
						api.UploadSecretLabel: "remotesecret",
					},
					Annotations: map[string]string{
						api.RemoteSecretNameAnnotation:          "rs",
						api.RemoteSecretPartialUpdateAnnotation: "true",
						api.RemoteSecretDeletedKeysAnnotation:   "b",
					},
				},
				Type: corev1.SecretTypeSSHAuth,
				Data: map[string][]byte{
					"a":                      []byte("va_new"),
					corev1.SSHAuthPrivateKey: []byte("vsshkey"),
				},
			})).To(Succeed())

			testSetup.SettleWithCluster(ITest.Context, func(g Gomega) {
				rs := *crenv.First[*api.RemoteSecret](&testSetup.InCluster)
				data, err := ITest.Storage.Get(ITest.Context, rs)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(*data).To(HaveLen(2))
				g.Expect((*data)["a"]).To(Equal([]byte("va_new")))
				g.Expect((*data)[corev1.SSHAuthPrivateKey]).To(Equal([]byte("vsshkey")))
			})
		})
	})
})
