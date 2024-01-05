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

	"github.com/prometheus/client_golang/prometheus"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func SetRemoteSecretCondition(ctx context.Context, rs *api.RemoteSecret, condition metav1.Condition) {
	currentCond := meta.FindStatusCondition(rs.Status.Conditions, condition.Type)
	meta.SetStatusCondition(&rs.Status.Conditions, condition)

	lg := log.FromContext(ctx)
	lg.Info("SetRemoteSecretCondition", "name", rs.Name, "namespace", rs.Namespace, "condition", condition, "currentCond", currentCond)
	if currentCond != nil {
		// Just set metrics if the status of the condition doesn't change.
		if currentCond.Status == condition.Status &&
			currentCond.Reason == condition.Reason && currentCond.Message == condition.Message {
			UpdateRemoteSecretConditionMetric(ctx, rs, &condition, 1.0)
			return
		}
		// Set previous condition to 0.0 if Status is changed.
		UpdateRemoteSecretConditionMetric(ctx, rs, currentCond, 0.0)
	}
	// Set current condition metric to 1.0 if Status is changed.
	UpdateRemoteSecretConditionMetric(ctx, rs, &condition, 1.0)
}
func DeleteRemoteSecretCondition(ctx context.Context, name, namespace string) {
	lg := log.FromContext(ctx)
	lg.Info("DeleteRemoteSecretCondition", "name", name, "namespace", namespace)
	RemoteSecretCondition.DeletePartialMatch(prometheus.Labels{"name": name, "namespace": namespace})

}

func UpdateRemoteSecretConditionMetric(ctx context.Context, rs *api.RemoteSecret, condition *metav1.Condition, value float64) {
	lg := log.FromContext(ctx)
	lg.Info("UpdateRemoteSecretConditionMetric", "name", rs.Name, "namespace", rs.Namespace, "condition", condition.Type, "status", string(condition.Status), "value", value)
	RemoteSecretCondition.WithLabelValues(rs.Name, rs.Namespace, condition.Type, string(condition.Status)).Set(value)
}
