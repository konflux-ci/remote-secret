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
	"errors"
	"fmt"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type RemoteSecretValidator struct{}

var (
	errGotNonSecret     = errors.New("RemoteSecret expected but got another type")
	errTargetsNotUnique = errors.New("targets are not unique in remote secret")
)

// +kubebuilder:webhook:path=/validate-appstudio-redhat-com-v1beta1-remotesecret,mutating=false,failurePolicy=fail,sideEffects=None,groups=appstudio.redhat.com,resources=remotesecrets,verbs=create;update,versions=v1beta1,name=mremotesecret.kb.io,admissionReviewVersions=v1
var _ webhook.CustomValidator = &RemoteSecretValidator{}

func (a *RemoteSecretValidator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	rs, ok := obj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("%w: %T", errGotNonSecret, obj)
	}
	return validateUniqueTargets(rs)
}

func (a *RemoteSecretValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) error {
	rs, ok := newObj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("%w: %T", errGotNonSecret, newObj)
	}
	return validateUniqueTargets(rs)
}

func (a *RemoteSecretValidator) ValidateDelete(_ context.Context, _ runtime.Object) error {
	return nil
}

func validateUniqueTargets(rs *v1beta1.RemoteSecret) error {
	targets := make(map[v1beta1.RemoteSecretTarget]string, len(rs.Spec.Targets))
	for _, t := range rs.Spec.Targets {
		if _, present := targets[t]; present {
			return fmt.Errorf("%w %s: %s", errTargetsNotUnique, rs.Name, rs.Spec.Targets)
		} else {
			targets[t] = ""
		}
	}
	return nil
}
