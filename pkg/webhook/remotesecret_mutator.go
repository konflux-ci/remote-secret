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
	"fmt"

	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"

	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type RemoteSecretMutator struct {
	Storage remotesecretstorage.RemoteSecretStorage
}

// +kubebuilder:webhook:path=/mutate-appstudio-redhat-com-v1beta1-remotesecret,mutating=true,failurePolicy=fail,sideEffects=None,groups=appstudio.redhat.com,resources=remotesecrets,verbs=create;update,versions=v1beta1,name=mremotesecret.kb.io,admissionReviewVersions=v1
var _ webhook.CustomDefaulter = &RemoteSecretMutator{}

func (a *RemoteSecretMutator) Default(ctx context.Context, obj runtime.Object) error {
	rs, ok := obj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("%w: %T", errGotNonSecret, obj)
	}
	auditLog := logs.AuditLog(ctx).WithValues("remoteSecret", client.ObjectKeyFromObject(rs))

	if len(rs.UploadData) != 0 {
		auditLog.Info("webhook data upload initiated")
		binData := map[string][]byte{}
		for k, v := range rs.UploadData {
			binData[k] = []byte(v)
		}

		err := a.Storage.Store(ctx, rs, &binData)
		if err != nil {
			err = fmt.Errorf("storage error on data save: %w", err)
			auditLog.Error(err, "webhook data upload failed")
			return err
		}

		auditLog.Info("webhook data upload completed")

		// clean upload data
		rs.UploadData = map[string]string{}
	}

	return nil
}
