package common

import (
	"fmt"
	esframework "github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/redhat-appstudio/remote-secret-e2e/framework"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	findValue = "{\"foo1\":\"foo1-val\"}"
)

func FindByName(f *esframework.Framework) (string, func(*framework.RsTestCase)) {
	return "[common] should find secrets by name using .DataFrom[]", func(tc *framework.RsTestCase) {
		const namePrefix = "e2e_find_name_%s_%s"
		secretKeyOne := fmt.Sprintf(namePrefix, f.Namespace.Name, "one")
		secretKeyTwo := fmt.Sprintf(namePrefix, f.Namespace.Name, "two")
		secretKeyThree := fmt.Sprintf(namePrefix, f.Namespace.Name, "three")
		secretValue := findValue
		tc.Secrets = map[string]esframework.SecretEntry{
			f.MakeRemoteRefKey(secretKeyOne):   {Value: secretValue},
			f.MakeRemoteRefKey(secretKeyTwo):   {Value: secretValue},
			f.MakeRemoteRefKey(secretKeyThree): {Value: secretValue},
		}
		tc.ExpectedSecret = &v1.Secret{
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				secretKeyOne:   []byte(secretValue),
				secretKeyTwo:   []byte(secretValue),
				secretKeyThree: []byte(secretValue),
			},
		}
		tc.RemoteSecret = &api.RemoteSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-remote-secret",
				Namespace: "default",
			},
		}
	}
}
