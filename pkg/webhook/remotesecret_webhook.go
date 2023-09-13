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
	"encoding/json"
	"net/http"

	"github.com/go-test/deep"
	adm "k8s.io/api/admission/v1"
	wh "sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
)

// +kubebuilder:webhook:path=/mutate-appstudio-redhat-com-v1beta1-remotesecret,mutating=true,failurePolicy=fail,sideEffects=None,groups=appstudio.redhat.com,resources=remotesecrets,verbs=create;update,versions=v1beta1,name=mremotesecret.kb.io,admissionReviewVersions=v1
type RemoteSecretWebhook struct {
	Validator WebhookValidator
	Mutator   WebhookMutator
	decoder   *wh.Decoder
}

// Handle implements admission.Handler.
func (w *RemoteSecretWebhook) Handle(ctx context.Context, req wh.Request) wh.Response {
	rs := &api.RemoteSecret{}
	obj := req.Object
	if req.Operation == adm.Delete {
		obj = req.OldObject
	}

	if err := w.decoder.DecodeRaw(obj, rs); err != nil {
		return wh.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case adm.Create:
		return w.handleCreate(ctx, req, rs)
	case adm.Update:
		old := &api.RemoteSecret{}
		if err := w.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return wh.Errored(http.StatusBadRequest, err)
		}
		return w.handleUpdate(ctx, req, old, rs)
	case adm.Delete:
		return w.handleDelete(ctx, req, rs)
	}

	return wh.Allowed("")
}

func (w *RemoteSecretWebhook) handleCreate(ctx context.Context, req wh.Request, rs *api.RemoteSecret) wh.Response {
	orig := rs.DeepCopy()
	if err := w.Validator.ValidateCreate(ctx, rs); err != nil {
		return wh.Denied(err.Error())
	}
	if err := w.Mutator.StoreUploadData(ctx, rs); err != nil {
		return wh.Denied(err.Error())
	}
	if err := w.Mutator.CopyDataFrom(ctx, req.UserInfo, rs); err != nil {
		return wh.Denied(err.Error())
	}
	return patchedOrAllowed(orig, req.Object.Raw, rs)
}

func (w *RemoteSecretWebhook) handleUpdate(ctx context.Context, req wh.Request, old *api.RemoteSecret, rs *api.RemoteSecret) wh.Response {
	orig := rs.DeepCopy()
	if err := w.Validator.ValidateUpdate(ctx, old, rs); err != nil {
		return wh.Denied(err.Error())
	}
	if err := w.Mutator.StoreUploadData(ctx, rs); err != nil {
		return wh.Denied(err.Error())
	}
	if err := w.Mutator.CopyDataFrom(ctx, req.UserInfo, rs); err != nil {
		return wh.Denied(err.Error())
	}
	return patchedOrAllowed(orig, req.Object.Raw, rs)
}

func (w *RemoteSecretWebhook) handleDelete(ctx context.Context, req wh.Request, rs *api.RemoteSecret) wh.Response {
	return wh.Allowed("")
}

// InjectDecoder implements admission.DecoderInjector.
func (w *RemoteSecretWebhook) InjectDecoder(decoder *wh.Decoder) error {
	w.decoder = decoder
	return nil
}

func patchedOrAllowed(orig any, origRaw []byte, obj any) wh.Response {
	if len(deep.Equal(orig, obj)) > 0 {
		json, err := json.Marshal(obj)
		if err != nil {
			return wh.Errored(http.StatusInternalServerError, err)
		}
		return wh.PatchResponseFromRaw(origRaw, json)
	}
	return wh.Allowed("")
}

var (
	_ wh.DecoderInjector = (*RemoteSecretWebhook)(nil)
	_ wh.Handler         = (*RemoteSecretWebhook)(nil)
)
