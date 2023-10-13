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

	"k8s.io/apimachinery/pkg/api/meta"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
)

type RemoteSecretValidator struct{}

var (
	errTargetsNotUnique                            = errors.New("targets are not unique in the remote secret")
	errDataFromSpecifiedWhenDataAlreadyPresent     = errors.New("dataFrom is not supported if there is data already present in the remote secret")
	errOnlyOneOfDataFromOrUploadDataCanBeSpecified = errors.New("only one of dataFrom or data can be specified")
)

// WebhookValidator defines the contract between the RemoteSecretWebhook and the "thing" that
// validates the remote secret passed to its methods. This interface mainly exists to ease the testing
// because it will only ever have one implementation in the production code - the RemoteSecretValidator.
type WebhookValidator interface {
	ValidateCreate(context.Context, *api.RemoteSecret) error
	ValidateUpdate(context.Context, *api.RemoteSecret, *api.RemoteSecret) error
	ValidateDelete(context.Context, *api.RemoteSecret) error
}

var _ WebhookValidator = (*RemoteSecretValidator)(nil)

func (a *RemoteSecretValidator) ValidateCreate(_ context.Context, rs *api.RemoteSecret) error {
	if err := validateUploadDataAndDataFrom(rs); err != nil {
		return err
	}
	return validateUniqueTargets(rs)
}

func (a *RemoteSecretValidator) ValidateUpdate(_ context.Context, _, new *api.RemoteSecret) error {
	if err := validateUploadDataAndDataFrom(new); err != nil {
		return err
	}
	if err := validateDataFrom(new); err != nil {
		return err
	}
	return validateUniqueTargets(new)
}

func (a *RemoteSecretValidator) ValidateDelete(_ context.Context, _ *api.RemoteSecret) error {
	return nil
}

func validateUniqueTargets(rs *api.RemoteSecret) error {
	targets := make(map[api.TargetKey]struct{}, len(rs.Spec.Targets))
	for _, t := range rs.Spec.Targets {
		tk := t.ToTargetKey(rs)
		if _, present := targets[tk]; present {
			return fmt.Errorf("%w", errTargetsNotUnique)
		} else {
			targets[tk] = struct{}{}
		}
	}
	return nil
}

func validateDataFrom(rs *api.RemoteSecret) error {
	var empty api.RemoteSecretDataFrom
	if rs.DataFrom != empty && meta.IsStatusConditionTrue(rs.Status.Conditions, string(api.RemoteSecretConditionTypeDataObtained)) {
		return errDataFromSpecifiedWhenDataAlreadyPresent
	}
	return nil
}

func validateUploadDataAndDataFrom(rs *api.RemoteSecret) error {
	var emptyDataFrom api.RemoteSecretDataFrom

	if rs.DataFrom != emptyDataFrom && len(rs.UploadData) > 0 {
		return errOnlyOneOfDataFromOrUploadDataCanBeSpecified
	}
	return nil
}
