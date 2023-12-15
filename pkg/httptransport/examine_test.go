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

package httptransport

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// response is nil, so there is no way to close the body
//
//nolint:bodyclose
func TestExaminingRoundTripper_RoundTrip(t *testing.T) {
	req, err := http.NewRequestWithContext(context.TODO(), "GET", "https://over.the.rainbow", strings.NewReader(""))
	assert.NoError(t, err)

	examinerCalled := false
	tr := ExaminingRoundTripper{RoundTripper: FakeRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:     "status",
			StatusCode: 42,
			Request:    r,
		}, nil
	}), Examiner: RoundTripExaminerFunc(func(request *http.Request, response *http.Response) error {
		assert.Equal(t, "status", response.Status)
		assert.Equal(t, 42, response.StatusCode)
		assert.Equal(t, request, response.Request)
		examinerCalled = true
		return nil
	})}

	_, _ = tr.RoundTrip(req)

	assert.True(t, examinerCalled)
}
