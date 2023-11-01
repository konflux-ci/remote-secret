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
	registry := prometheus.NewPedanticRegistry()
	UploadRejectionsCounter.Reset()

	assert.NoError(t, RegisterCommonMetrics(registry))

	UploadRejectionsCounter.WithLabelValues("foo_operation", "some_reason").Inc()
	UploadRejectionsCounter.WithLabelValues("foo_operation", "some_other_reason").Inc()

	count, err := prometheusTest.GatherAndCount(registry, "redhat_appstudio_remotesecret_data_upload_rejected_total")
	assert.Equal(t, 2, count)
	assert.NoError(t, err)
}
