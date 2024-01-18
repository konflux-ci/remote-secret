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

package availability

import (
	"context"
	"time"

	rsmetrics "github.com/redhat-appstudio/remote-secret/pkg/metrics"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	ctrl "sigs.k8s.io/controller-runtime"
)

type SystemsWatchdog struct {
	SecretStorage secretstorage.SecretStorage
}

func (r *SystemsWatchdog) Start(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Second) //TODO: configurable?
	go func() {
		for {
			select {
			case <-ticker.C:
				r.checkStorage(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
	return nil
}

func (r *SystemsWatchdog) checkStorage(ctx context.Context) {
	if err := r.SecretStorage.Examine(ctx); err != nil {
		ctrl.Log.Error(err, "secret storage is not available")
		rsmetrics.AvailabilityGauge.Inc()
	} else {
		ctrl.Log.Info("secret storage is available")
		rsmetrics.AvailabilityGauge.Set(0)
	}
}
