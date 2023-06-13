package integrationtests

import (
	"context"

	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var ITest = struct {
	TestEnvironment       *envtest.Environment
	Context               context.Context //nolint: containedctx // we DO want the context shared across all integration tests...
	Cancel                context.CancelFunc
	Client                client.Client
	Storage               remotesecretstorage.RemoteSecretStorage
	OperatorConfiguration *config.OperatorConfiguration
}{}
