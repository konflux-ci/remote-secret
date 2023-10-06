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

package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	admissionv1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
)

func TestHandle_Create(t *testing.T) {
	mutator := &TestMutator{}
	validator := &TestValidator{}

	scheme := runtime.NewScheme()
	api.AddToScheme(scheme)

	decoder, err := admission.NewDecoder(scheme)
	assert.NoError(t, err)

	w := RemoteSecretWebhook{
		Validator: validator,
		Mutator:   mutator,
	}
	w.InjectDecoder(decoder)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name:      "rs",
			Namespace: "default",
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion": "appstudio.redhat.com/v1beta1", "kind": "RemoteSecret", "metadata": {"name": "rs", "namespace": "default"}}`),
			},
		},
	}

	validator.On("ValidateCreate", mock.Anything, mock.Anything).Return(nil)
	mutator.On("StoreUploadData", mock.Anything, mock.Anything).Return(nil)
	mutator.On("CopyDataFrom", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	res := w.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	validator.AssertCalled(t, "ValidateCreate", mock.Anything, mock.Anything)
	mutator.AssertCalled(t, "StoreUploadData", mock.Anything, mock.Anything)
	mutator.AssertCalled(t, "CopyDataFrom", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandle_Update(t *testing.T) {
	mutator := &TestMutator{}
	validator := &TestValidator{}

	scheme := runtime.NewScheme()
	api.AddToScheme(scheme)

	decoder, err := admission.NewDecoder(scheme)
	assert.NoError(t, err)

	w := RemoteSecretWebhook{
		Validator: validator,
		Mutator:   mutator,
	}
	w.InjectDecoder(decoder)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name:      "rs",
			Namespace: "default",
			Operation: admissionv1.Update,
			OldObject: runtime.RawExtension{
				Raw: []byte(`{"apiVersion": "appstudio.redhat.com/v1beta1", "kind": "RemoteSecret", "metadata": {"name": "rs", "namespace": "default"}}`),
			},
			Object: runtime.RawExtension{
				Raw: []byte(`{"apiVersion": "appstudio.redhat.com/v1beta1", "kind": "RemoteSecret", "metadata": {"name": "rs", "namespace": "default"}}`),
			},
		},
	}

	validator.On("ValidateUpdate", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mutator.On("StoreUploadData", mock.Anything, mock.Anything).Return(nil)
	mutator.On("CopyDataFrom", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	res := w.Handle(context.TODO(), req)

	assert.True(t, res.Allowed)
	validator.AssertCalled(t, "ValidateUpdate", mock.Anything, mock.Anything, mock.Anything)
	mutator.AssertCalled(t, "StoreUploadData", mock.Anything, mock.Anything)
	mutator.AssertCalled(t, "CopyDataFrom", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandle_Delete(t *testing.T) {
	scheme := runtime.NewScheme()
	api.AddToScheme(scheme)

	decoder, err := admission.NewDecoder(scheme)
	assert.NoError(t, err)

	w := RemoteSecretWebhook{}
	w.InjectDecoder(decoder)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name:      "rs",
			Namespace: "default",
			Operation: admissionv1.Delete,
			OldObject: runtime.RawExtension{
				Raw: []byte(`{"apiVersion": "appstudio.redhat.com/v1beta1", "kind": "RemoteSecret", "metadata": {"name": "rs", "namespace": "default"}}`),
			},
		},
	}

	res := w.Handle(context.TODO(), req)
	assert.True(t, res.Allowed)
}

type TestValidator struct {
	mock.Mock
}

type TestMutator struct {
	mock.Mock
}

// ValidateCreate implements WebhookValidator.
func (v *TestValidator) ValidateCreate(ctx context.Context, rs *api.RemoteSecret) error {
	args := v.Called(ctx, rs)
	return args.Error(0)
}

// ValidateDelete implements WebhookValidator.
func (v *TestValidator) ValidateDelete(ctx context.Context, rs *api.RemoteSecret) error {
	args := v.Called(ctx, rs)
	return args.Error(0)
}

// ValidateUpdate implements WebhookValidator.
func (v *TestValidator) ValidateUpdate(ctx context.Context, old *api.RemoteSecret, new *api.RemoteSecret) error {
	args := v.Called(ctx, old, new)
	return args.Error(0)
}

// CopyDataFrom implements WebhookMutator.
func (m *TestMutator) CopyDataFrom(ctx context.Context, user authv1.UserInfo, rs *api.RemoteSecret) error {
	args := m.Called(ctx, user, rs)
	return args.Error(0)
}

// StoreUploadData implements WebhookMutator.
func (m *TestMutator) StoreUploadData(ctx context.Context, rs *api.RemoteSecret) error {
	args := m.Called(ctx, rs)
	return args.Error(0)
}

var (
	_ WebhookValidator = (*TestValidator)(nil)
	_ WebhookMutator   = (*TestMutator)(nil)
)
