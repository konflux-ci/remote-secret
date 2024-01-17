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

package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
)

var UploadRejectionsCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: config.MetricsNamespace,
		Subsystem: config.MetricsSubsystem,
		Name:      "data_upload_rejected_total",
		Help:      "The number of remote secret uploads rejected by the webhook or controller",
	},
	[]string{"operation", "reason"},
)

var HealthCheckCounter = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: config.MetricsNamespace,
		Subsystem: config.MetricsSubsystem,
		Name:      "systems_available",
		Help:      "The availability of the remote secret system",
	})

func RegisterCommonMetrics(registerer prometheus.Registerer) error {
	if err := registerer.Register(UploadRejectionsCounter); err != nil {
		return fmt.Errorf("failed to register rejected uploads count metric: %w", err)
	}
	if err := registerer.Register(HealthCheckCounter); err != nil {
		return fmt.Errorf("failed to register health check metric: %w", err)
	}

	return nil
}
