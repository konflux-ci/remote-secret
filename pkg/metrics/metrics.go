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
	"context"

	"github.com/redhat-appstudio/remote-secret/pkg/logs"

	"github.com/prometheus/client_golang/prometheus"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

var RemoteSecretConditionGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: config.MetricsNamespace,
		Subsystem: config.MetricsSubsystem,
		Name:      "status_condition",
		Help:      "The status condition of a specific RemoteSecret",
	},
	[]string{"name", "namespace", "condition", "status"},
)

func RegisterCommonMetrics(registerer prometheus.Registerer) error {
	registerer.MustRegister(UploadRejectionsCounter, RemoteSecretConditionGauge)
	return nil
}

func DeleteRemoteSecretCondition(ctx context.Context, name, namespace string) {
	lg := log.FromContext(ctx)
	lg.V(logs.DebugLevel).Info("DeleteRemoteSecretCondition", "name", name, "namespace", namespace)
	RemoteSecretConditionGauge.DeletePartialMatch(prometheus.Labels{"name": name, "namespace": namespace})

}

func UpdateRemoteSecretConditionMetric(ctx context.Context, rs *api.RemoteSecret, condition *metav1.Condition, value float64) {
	lg := log.FromContext(ctx)
	lg.V(logs.DebugLevel).Info("UpdateRemoteSecretConditionMetric", "name", rs.Name, "namespace", rs.Namespace, "condition", condition.Type, "status", string(condition.Status), "value", value)
	RemoteSecretConditionGauge.WithLabelValues(rs.Name, rs.Namespace, condition.Type, string(condition.Status)).Set(value)
}
