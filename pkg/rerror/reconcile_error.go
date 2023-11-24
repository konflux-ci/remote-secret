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

package rerror

import (
	"strings"
)

type AggregatedError struct {
	errors []error
}

func AggregateNonNilErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}

	aggregated := AggregatedError{}
	for _, e := range errs {
		if e != nil {
			aggregated.Add(e)
		}
	}

	if len(aggregated.errors) == 0 {
		return nil
	}

	if len(aggregated.errors) == 1 {
		return aggregated.errors[0]
	} else {
		return &aggregated
	}
}

func NewAggregatedError(errs ...error) *AggregatedError {
	return &AggregatedError{
		errors: errs,
	}
}

func (ae *AggregatedError) Add(errs ...error) {
	ae.errors = append(ae.errors, errs...)
}

func (ae *AggregatedError) Error() string {
	strs := make([]string, len(ae.errors))
	for i, e := range ae.errors {
		strs[i] = e.Error()
	}

	return strings.Join(strs, ", ")
}

func (ae *AggregatedError) HasErrors() bool {
	return len(ae.errors) > 0
}

func (ae *AggregatedError) Unwrap() []error {
	return ae.errors
}
