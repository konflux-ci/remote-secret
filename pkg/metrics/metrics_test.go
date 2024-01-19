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

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	prometheusTest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRegisterMetrics(t *testing.T) {

	var tests = []struct {
		name          string
		resetFunc     func()
		incrementFunc func()
		metricName    string
		want          int
	}{
		{"rejection counter", func() {
			UploadRejectionsCounter.Reset()
		}, func() {
			UploadRejectionsCounter.WithLabelValues("foo_operation", "some_reason").Inc()
			UploadRejectionsCounter.WithLabelValues("foo_operation", "some_other_reason").Inc()
		}, "redhat_appstudio_remotesecret_data_upload_rejected_total", 2},
		{"condition gauge", func() {
			RemoteSecretConditionGauge.Reset()
		}, func() {
			RemoteSecretConditionGauge.WithLabelValues("DataObtained", "test-remote-secret", "default", "false").Inc()
		}, "redhat_appstudio_remotesecret_status_condition", 1},
	}
	// The execution loop
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewPedanticRegistry()
			tt.resetFunc()
			assert.NoError(t, RegisterCommonMetrics(registry))
			tt.incrementFunc()
			count, err := prometheusTest.GatherAndCount(registry, tt.metricName)
			assert.Equal(t, tt.want, count)
			assert.NoError(t, err)
		})
	}
}
