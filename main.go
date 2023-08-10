/*
Copyright 2021.

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

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redhat-appstudio/remote-secret/webhook"

	"github.com/alexflint/go-arg"
	"github.com/redhat-appstudio/remote-secret/controllers"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/pkg/cmd"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	corev1 "k8s.io/api/core/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(api.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	args := cmd.OperatorCliArgs{}
	arg.MustParse(&args)
	logs.InitLoggers(args.ZapDevel, args.ZapEncoder, args.ZapLogLevel, args.ZapStackTraceLevel, args.ZapTimeEncoding)

	var err error
	err = config.SetupCustomValidations(config.CustomValidationOptions{AllowInsecureURLs: args.AllowInsecureURLs})
	if err != nil {
		setupLog.Error(err, "failed to initialize the validators")
		os.Exit(1)
	}

	setupLog.Info("Starting remote secret operator with environment", "env", os.Environ(), "configuration", &args)

	ctx := ctrl.SetupSignalHandler()
	ctx = context.WithValue(ctx, config.InstanceIdContextKey, args.CommonCliArgs.InstanceId)
	ctx = log.IntoContext(ctx, ctrl.Log.WithValues("instanceId", args.CommonCliArgs.InstanceId))

	mgr, mgrErr := createManager(args)
	if mgrErr != nil {
		setupLog.Error(mgrErr, "unable to start manager")
		os.Exit(1)
	}

	cfg, err := LoadFrom(&args)
	if err != nil {
		setupLog.Error(err, "Failed to load the configuration")
		os.Exit(1)
	}

	secretStorage, err := cmd.CreateInitializedSecretStorage(ctx, &args.CommonCliArgs)
	if err != nil {
		setupLog.Error(err, "failed to initialize the secret storage")
		os.Exit(1)
	}

	cf := &bindings.CachingClientFactory{
		LocalCluster: bindings.LocalClusterConnectionDetails{
			Client: mgr.GetClient(),
			Config: mgr.GetConfig(),
		},
	}

	if err = controllers.SetupAllReconcilers(mgr, &cfg, secretStorage, cf); err != nil {
		setupLog.Error(err, "failed to set up the controllers")
		os.Exit(1)
	}

	rs := &api.RemoteSecret{}
	err = ctrl.NewWebhookManagedBy(mgr).
		WithDefaulter(&webhook.RemoteSecretMutator{Storage: secretStorage}).
		WithValidator(&webhook.RemoteSecretValidator{}).
		For(rs).
		Complete()
	if err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "RemoteSecret")
		os.Exit(1)
	}
	/////////////////////

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func LoadFrom(args *cmd.OperatorCliArgs) (config.OperatorConfiguration, error) {
	ret := config.OperatorConfiguration{EnableRemoteSecrets: args.EnableRemoteSecrets, EnableTokenUpload: args.EnableRemoteSecrets}
	return ret, nil
}

func createManager(args cmd.OperatorCliArgs) (manager.Manager, error) {
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     args.MetricsAddr,
		HealthProbeBindAddress: args.ProbeAddr,
		LeaderElection:         args.EnableLeaderElection,
		LeaderElectionID:       "4279163b.appstudio.redhat.org",
		Logger:                 ctrl.Log,
	}
	restConfig := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(restConfig, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager %w", err)
	}

	return mgr, nil
}
