//
// Copyright (c) 2021 Red Hat, Inc.
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

package controllers

import (
	"context"
	"fmt"

	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func SetupAllReconcilers(mgr controllerruntime.Manager, cfg *config.OperatorConfiguration, secretStorage secretstorage.SecretStorage, cf bindings.ClientFactory) error {
	ctx := context.Background()

	remoteSecretStorage := remotesecretstorage.NewJSONSerializingRemoteSecretStorage(secretStorage)
	if err := remoteSecretStorage.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize the remote secret storage: %w", err)
	}

	if err := (&RemoteSecretReconciler{
		Client:              mgr.GetClient(),
		TargetClientFactory: cf,
		Scheme:              mgr.GetScheme(),
		Configuration:       cfg,
		RemoteSecretStorage: remoteSecretStorage,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := remoteSecretStorage.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize the remote secret storage: %w", err)
	}

	if err := (&TokenUploadReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		RemoteSecretStorage: remoteSecretStorage,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
