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

package webhook

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
)

// TODO REMOVE IT
type MemoryStorage2 struct {
	// Data is the map of stored data.
	Data map[SecretID2][]byte
	// ErrorOnInitialize if not nil, the error is thrown when the Initialize method is called.
	ErrorOnInitialize error
	// ErrorOnStore if not nil, the error is thrown when the Store method is called.
	ErrorOnStore error
	// ErrorOnGet if not nil, the error is thrown when the Get method is called.
	ErrorOnGet error
	// ErrorOnDelete if not nil, the error is thrown when the Delete method is called.
	ErrorOnDelete error

	lock sync.RWMutex

	lg logr.Logger
}

// Delete implements secretstorage.SecretStorage
func (m *MemoryStorage2) Delete(ctx context.Context, id SecretID2) error {
	if m.ErrorOnDelete != nil {
		return m.ErrorOnDelete
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	m.ensureTokens()

	delete(m.Data, id)
	return nil
}

// Get implements secretstorage.SecretStorage
func (m *MemoryStorage2) Get(ctx context.Context, id SecretID2) ([]byte, error) {
	if m.ErrorOnGet != nil {
		return nil, m.ErrorOnGet
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	m.ensureTokens()

	data, ok := m.Data[id]
	if !ok {
		return nil, fmt.Errorf("%w", secretstorage.NotFoundError)
	}

	return data, nil
}

// Initialize implements secretstorage.SecretStorage
func (m *MemoryStorage2) Initialize(ctx context.Context) error {
	if m.ErrorOnInitialize != nil {
		return m.ErrorOnInitialize
	}

	m.lg = log.FromContext(ctx)
	m.lock.Lock()
	defer m.lock.Unlock()

	m.Data = map[SecretID2][]byte{}
	return nil
}

// Store implements secretstorage.SecretStorage
func (m *MemoryStorage2) Store(ctx context.Context, id SecretID2, data []byte) error {
	if m.ErrorOnStore != nil {
		return m.ErrorOnStore
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	m.ensureTokens()

	m.Data[id] = data
	m.lg.Info("memory storage stored: ", "id", id, "len(secretData)", fmt.Sprint(len(data)))

	return nil
}

func (m *MemoryStorage2) ensureTokens() {
	if m.Data == nil {
		m.Data = map[SecretID2][]byte{}
	}
}

var _ SecretStorage2 = (*MemoryStorage2)(nil)
