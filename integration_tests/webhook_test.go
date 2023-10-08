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

// tests for validating and mutating webhooks
import (
	"github.com/metlos/crenv"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
)

var _ = Describe("MutatorTest", func() {
	Describe("Upload remote secret", func() {
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
					UploadData: map[string][]byte{
						"test":  []byte("test1"),
						"test2": []byte("test2"),
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

var _ = Describe("ValidatorTest", func() {
	Describe("Upload valid remote secret", func() {
		When("RemoteSecret with different targets", func() {
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

			It("should validate ok", func() {
				rs := &api.RemoteSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-remote-secret",
						Namespace: "default",
					},
					Spec: api.RemoteSecretSpec{
						Targets: []api.RemoteSecretTarget{
							{Namespace: "ns1", ApiUrl: "https://test1.com"},
							{Namespace: "ns2", ApiUrl: "https://test2.com"},
						},
					},
					UploadData: map[string][]byte{
						"test": []byte("test1"),
					},
				}
				err := ITest.Client.Create(ITest.Context, rs)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Upload malformed remote secret", func() {
		When("RemoteSecret with duplicate targets", func() {
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

			It("should throw error on remote secret creation", func() {
				rs := &api.RemoteSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-remote-secret",
						Namespace: "default",
					},
					Spec: api.RemoteSecretSpec{
						Targets: []api.RemoteSecretTarget{
							{Namespace: "ns1", ApiUrl: "https://test.com", ClusterCredentialsSecret: "abc##"},
							{Namespace: "ns1", ApiUrl: "https://test.com", ClusterCredentialsSecret: "abc##"},
						},
					},
					UploadData: map[string][]byte{
						"test": []byte("test1"),
					},
				}
				err := ITest.Client.Create(ITest.Context, rs)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("targets are not unique in the remote secret"))
			})
		})
	})
})
