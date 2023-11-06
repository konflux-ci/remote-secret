/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package vault

import (
	esoframework "github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/redhat-appstudio/remote-secret-e2e/framework"
	"github.com/redhat-appstudio/remote-secret-e2e/suites/provider/cases/common"

	// nolint
	. "github.com/onsi/ginkgo/v2"
)

const (
	withTokenAuth    = "with token auth"
	withCertAuth     = "with cert auth"
	withApprole      = "with approle auth"
	withV1           = "with v1 provider"
	withJWT          = "with jwt provider"
	withJWTK8s       = "with jwt k8s provider"
	withK8s          = "with kubernetes provider"
	withReferentAuth = "with referent provider"
)

var _ = Describe("[remote-secrets][vault]", Label("vault", "remote-secret"), func() {
	f := esoframework.New("remote-secrets-vault")
	prov := newVaultProvider(f)

	DescribeTable("sync secrets",
		framework.TableFunc(f, prov),
		// uses token auth
		framework.Compose(withTokenAuth, f, common.FindByName, noOp),
		// use cert auth
		framework.Compose(withCertAuth, f, common.FindByName, noOp),
		// use approle auth
		framework.Compose(withApprole, f, common.FindByName, noOp),
		// use jwt provider
		framework.Compose(withJWT, f, common.FindByName, noOp),

		// use kubernetes provider
		framework.Compose(withK8s, f, common.FindByName, noOp),
	)
})

func noOp(tc *framework.RsTestCase) {
}
