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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var Error1 = errors.New("1")
var Error2 = errors.New("2")
var ErrorA = errors.New("a")
var ErrorB = errors.New("b")
var ErrorC = errors.New("c")
var RandomError = errors.New("asdf")

func TestAggregatedError(t *testing.T) {
	e := NewAggregatedError(ErrorA, ErrorB, ErrorC)
	assert.Equal(t, "a, b, c", e.Error())
}

func TestAggregatedError_Add(t *testing.T) {
	e := NewAggregatedError()

	assert.Equal(t, "", e.Error())

	e.Add(ErrorA)
	assert.Equal(t, "a", e.Error())

	e.Add(ErrorB)
	assert.Equal(t, "a, b", e.Error())

	e.Add(ErrorC)
	assert.Equal(t, "a, b, c", e.Error())
}

func TestAggregateNonNilErrors(t *testing.T) {
	t.Run("single nil doesn't count", func(t *testing.T) {
		assert.Nil(t, AggregateNonNilErrors(nil))
	})

	t.Run("all nils don't count", func(t *testing.T) {
		assert.Nil(t, AggregateNonNilErrors(nil, nil, nil))
	})

	t.Run("single error returned as is", func(t *testing.T) {
		err := RandomError
		assert.Same(t, err, AggregateNonNilErrors(err))
	})

	t.Run("single error among nils returned as is", func(t *testing.T) {
		err := RandomError
		assert.Same(t, err, AggregateNonNilErrors(nil, err, nil, nil))
	})

	t.Run("multiple errors aggregated", func(t *testing.T) {
		err1 := Error1
		err2 := Error2
		agg := AggregateNonNilErrors(err1, err2)
		assert.IsType(t, &AggregatedError{}, agg)
		assert.Equal(t, "1, 2", agg.Error())
	})
}
