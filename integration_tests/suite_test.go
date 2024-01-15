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
	"context"
	"path/filepath"
	"testing"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/redhat-appstudio/remote-secret/pkg/webhook"
	crwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-appstudio/remote-secret/controllers"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Remote Secret Operator Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logs.InitDevelLoggers()
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ITest.Context, ITest.Cancel = context.WithCancel(context.TODO())

	By("bootstrapping the test environment")
	ITest.TestEnvironment = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "config", "webhook", "base")},
		},
	}

	cfg, err := ITest.TestEnvironment.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = api.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	ITest.Client, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	ITest.Storage = newITestStorage()
	//ITest.MemoryStorage = &memorystorage.MemoryStorage{}
	//ITest.Storage = remotesecretstorage.NewJSONSerializingRemoteSecretStorage(ITest.MemoryStorage)

	Expect(ITest.Storage.Initialize(ITest.Context)).To(Succeed())

	webhookInstallOptions := &ITest.TestEnvironment.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: crwebhook.NewServer(crwebhook.Options{
			Port:    webhookInstallOptions.LocalServingPort,
			Host:    webhookInstallOptions.LocalServingHost,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),

		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	ITest.OperatorConfiguration = &config.OperatorConfiguration{}

	ITest.ClientFactory = TestClientFactory{
		GetClientImpl: func(_ context.Context, _ string, _ *api.RemoteSecretTarget, _ *api.TargetStatus) (client.Client, error) {
			// effectively, this switches off any support for auth or deployment to remote clusters in the integration tests..
			return mgr.GetClient(), nil
		},
	}

	Expect(controllers.SetupAllReconcilers(mgr, ITest.OperatorConfiguration, ITest.Storage.SecretStorage(), &ITest.ClientFactory)).To(Succeed())
	Expect(webhook.SetupAllWebhooks(mgr, ITest.Storage.SecretStorage())).To(Succeed())

	go func() {
		err = mgr.Start(ITest.Context)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()
}, 3600)

var _ = AfterSuite(func() {
	if ITest.Cancel != nil {
		ITest.Cancel()
	}

	By("tearing down the test environment")
	if ITest.TestEnvironment != nil {
		err := ITest.TestEnvironment.Stop()
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = BeforeEach(func() {
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>", "test", CurrentGinkgoTestDescription().FullTestText)
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>")
	log.Log.Info(">>>>>>>")
})

var _ = AfterEach(func() {
	testDesc := CurrentGinkgoTestDescription()
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<", "test", testDesc.FullTestText, "duration", testDesc.Duration, "failed", testDesc.Failed)
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<")
	log.Log.Info("<<<<<<<", "memory storage len", ITest.Storage.Len())
	log.Log.Info("<<<<<<<")
	ITest.Storage.Reset()
	Eventually(func(g Gomega) int {
		return ITest.Storage.Len()
	}).Should(Equal(0))

})
