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
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/go-logr/logr"
	rsmetrics "github.com/redhat-appstudio/remote-secret/pkg/metrics"
	"github.com/redhat-appstudio/remote-secret/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	crebhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/pkg/cmd"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
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

	mgr, mgrErr := createManager(setupLog, args)
	if mgrErr != nil {
		setupLog.Error(mgrErr, "unable to start manager")
		os.Exit(1)
	}

	cfg, err := LoadFrom(&args)
	if err != nil {
		setupLog.Error(err, "Failed to load the configuration")
		os.Exit(1)
	}

	secretStorage, err := cmd.CreateInitializedSecretStorage(ctx, mgr.GetClient(), &args.CommonCliArgs)
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

	if err = rsmetrics.RegisterCommonMetrics(metrics.Registry); err != nil {
		setupLog.Error(err, "failed to register common metrics")
		os.Exit(1)
	}

	if err = controllers.SetupAllReconcilers(mgr, &cfg, secretStorage, cf); err != nil {
		setupLog.Error(err, "failed to set up the controllers")
		os.Exit(1)
	}

	if err = webhook.SetupAllWebhooks(mgr, secretStorage); err != nil {
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

	if args.ExposeProfiling {
		// This can be replaced by mgr.PprofBindAddress when we finally upgrade to controller-runtime 0.15.x
		if err := mgr.AddMetricsExtraHandler("/debug/pprof/", http.HandlerFunc(pprof.Index)); err != nil {
			setupLog.Error(err, "failed to set up the profiling endpoint")
			os.Exit(1)
		}
		if err := mgr.AddMetricsExtraHandler("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline)); err != nil {
			setupLog.Error(err, "failed to set up the profiling endpoint")
			os.Exit(1)
		}
		if err := mgr.AddMetricsExtraHandler("/debug/pprof/profile", http.HandlerFunc(pprof.Profile)); err != nil {
			setupLog.Error(err, "failed to set up the profiling endpoint")
			os.Exit(1)
		}
		if err := mgr.AddMetricsExtraHandler("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol)); err != nil {
			setupLog.Error(err, "failed to set up the profiling endpoint")
			os.Exit(1)
		}
		if err := mgr.AddMetricsExtraHandler("/debug/pprof/trace", http.HandlerFunc(pprof.Trace)); err != nil {
			setupLog.Error(err, "failed to set up the profiling endpoint")
			os.Exit(1)
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func LoadFrom(args *cmd.OperatorCliArgs) (config.OperatorConfiguration, error) {
	ret := config.OperatorConfiguration{}
	return ret, nil
}

func createManager(lg logr.Logger, args cmd.OperatorCliArgs) (manager.Manager, error) {

	restConfig := ctrl.GetConfigOrDie()
	disableHTTP2 := func(c *tls.Config) {
		if !args.DisableHTTP2 {
			return
		}
		lg.Info("Disabling HTTP/2")
		c.NextProtos = []string{"http/1.1"}
	}

	webhookServerOptions := crebhook.Options{
		TLSOpts: []func(config *tls.Config){disableHTTP2},
	}

	webhookServer := crebhook.NewServer(webhookServerOptions)
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     args.MetricsAddr,
		HealthProbeBindAddress: args.ProbeAddr,
		LeaderElection:         args.EnableLeaderElection,
		LeaderElectionID:       "4279163b.appstudio.redhat.org",
		Logger:                 ctrl.Log,
		WebhookServer:          webhookServer,
	}

	mgr, err := ctrl.NewManager(restConfig, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager %w", err)
	}

	return mgr, nil
}
