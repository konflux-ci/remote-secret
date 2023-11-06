package framework

import (
	"context"

	"github.com/external-secrets/external-secrets-e2e/framework"
	. "github.com/onsi/gomega"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RsTestCase contains the test infra to run a table driven test.
type RsTestCase struct {
	Framework         *framework.Framework
	RemoteSecret      *api.RemoteSecret
	AdditionalObjects []client.Object
	Secrets           map[string]framework.SecretEntry
	ExpectedSecret    *v1.Secret
	AfterSync         func(framework.SecretStoreProvider, *v1.Secret)
}

// TableFunc returns the main func that runs a TestCase in a table driven test.
func TableFunc(f *framework.Framework, prov framework.SecretStoreProvider) func(...func(*RsTestCase)) {
	return func(tweaks ...func(*RsTestCase)) {
		var err error

		// make default test case
		// and apply customization to it
		tc := makeDefaultTestCase(f)
		for _, tweak := range tweaks {
			tweak(tc)
		}

		// create secrets & defer delete
		for k, v := range tc.Secrets {
			key := k
			prov.CreateSecret(key, v)
			defer func() {
				prov.DeleteSecret(key)
			}()
		}
		ctx := context.Background()
		// create v1beta1 external secret otherwise
		err = tc.Framework.CRClient.Create(ctx, tc.RemoteSecret)
		Expect(err).ToNot(HaveOccurred())

		if tc.AdditionalObjects != nil {
			for _, obj := range tc.AdditionalObjects {
				err = tc.Framework.CRClient.Create(ctx, obj)
				Expect(err).ToNot(HaveOccurred())
			}
		}

		// wait for Kind=Secret to have the expected data
		secret, err := tc.Framework.WaitForSecretValue(tc.Framework.Namespace.Name, tc.RemoteSecret.Spec.Secret.Name, tc.ExpectedSecret)
		if err != nil {
			log.FromContext(ctx).Info("Did not match.", "Expected", tc.ExpectedSecret, "Got", secret)
		}

		Expect(err).ToNot(HaveOccurred())
		tc.AfterSync(prov, secret)
	}
}

func makeDefaultTestCase(f *framework.Framework) *RsTestCase {
	return &RsTestCase{
		AfterSync: func(ssp framework.SecretStoreProvider, s *v1.Secret) {},
		Framework: f,
		RemoteSecret: &api.RemoteSecret{
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
	}
}
