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
package e2e

import (
	"testing"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"

	esoaddon "github.com/external-secrets/external-secrets-e2e/framework/addon"
	"github.com/external-secrets/external-secrets-e2e/framework/util"
	"github.com/redhat-appstudio/remote-secret-e2e/framework/addon"
	_ "github.com/redhat-appstudio/remote-secret-e2e/suites/provider/cases"
)

var _ = SynchronizedBeforeSuite(func() []byte {
	cfg := &esoaddon.Config{}
	cfg.KubeConfig, cfg.KubeClientSet, cfg.CRClient = util.NewConfig()

	//By("installing eso")
	//	esoaddon.InstallGlobalAddon(esoaddon.NewESO(esoaddon.WithCRDs()), cfg)

	By("installing rs")
	esoaddon.InstallGlobalAddon(addon.NewRemoteSecretDeployment(cfg), cfg)

	return nil
}, func([]byte) {
	// noop
})

var _ = SynchronizedAfterSuite(func() {
	// noop
}, func() {
	By("Cleaning up global addons")
	esoaddon.UninstallGlobalAddons()
	if CurrentSpecReport().Failed() {
		esoaddon.PrintLogs()
	}
})

func TestRsE2E(t *testing.T) {
	NewWithT(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e suite", Label("remote-secrets"))
}
