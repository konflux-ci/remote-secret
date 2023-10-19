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

	"github.com/redhat-appstudio/remote-secret/pkg/metrics"
	authv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
)

var errorCopyNotAllowed = errors.New("user cannot copy the data of the specified remote secret")

var metricUploadDataOperationLabel = "webhook_data_upload"
var metricCopyDataDataOperationLabel = "copy_data_from"

// +kubebuilder:rbac:groups="authorization.k8s.io",resources=subjectaccessreviews,verbs=create

// WebhookMutator defines the contract between the RemoteSecretWebhook and the "thing" that
// mutates the remote secret passed to its methods. This interface mainly exists to ease the testing
// because it will only ever have one implementation in the production code - the RemoteSecretMutator.
type WebhookMutator interface {
	StoreUploadData(context.Context, *api.RemoteSecret) error
	CopyDataFrom(context.Context, authv1.UserInfo, *api.RemoteSecret) error
}

type RemoteSecretMutator struct {
	Client  client.Client
	Storage remotesecretstorage.RemoteSecretStorage
}

var _ WebhookMutator = (*RemoteSecretMutator)(nil)

func (m *RemoteSecretMutator) StoreUploadData(ctx context.Context, rs *api.RemoteSecret) error {
	binData := rs.UploadData

	if len(rs.StringUploadData) > 0 {
		if binData == nil {
			binData = make(map[string][]byte, len(rs.StringUploadData))
		}
		for k, v := range rs.StringUploadData {
			binData[k] = []byte(v)
		}
	}

	if len(binData) > 0 {
		auditLog := logs.AuditLog(ctx).WithValues("remoteSecret", client.ObjectKeyFromObject(rs))
		auditLog.Info("webhook data upload initiated")

		err := m.Storage.Store(ctx, rs, &binData)
		if err != nil {
			metrics.UploadRejectionsCounter.WithLabelValues(metricUploadDataOperationLabel, "storage_write_failed").Inc()
			err = fmt.Errorf("storage error on data save: %w", err)
			auditLog.Error(err, "webhook data upload failed")
			return err
		}

		auditLog.Info("webhook data upload completed")
	}

	// clean upload data
	rs.UploadData = nil
	rs.StringUploadData = nil

	return nil
}

func (m *RemoteSecretMutator) CopyDataFrom(ctx context.Context, user authv1.UserInfo, rs *api.RemoteSecret) error {
	if rs.DataFrom.Name == "" {
		return nil
	}

	sourceName := rs.DataFrom.Name
	sourceNamespace := rs.DataFrom.Namespace
	if sourceNamespace == "" {
		sourceNamespace = rs.Namespace
	}

	if err := m.checkHasPermissions(ctx, user, sourceName, sourceNamespace); err != nil {
		metrics.UploadRejectionsCounter.WithLabelValues(metricCopyDataDataOperationLabel, "insufficient_permissions").Inc()
		if errors.Is(err, errorCopyNotAllowed) {
			return fmt.Errorf("%w", err)
		}
		return fmt.Errorf("failed to check the permissions of remote secret %s in namespace %s for user %s: %w", sourceName, sourceNamespace, user.Username, err)
	}

	source := &api.RemoteSecret{}
	if err := m.Client.Get(ctx, client.ObjectKey{Name: sourceName, Namespace: sourceNamespace}, source); err != nil {
		metrics.UploadRejectionsCounter.WithLabelValues(metricUploadDataOperationLabel, "source_not_found").Inc()
		return fmt.Errorf("failed to get the source remote secret for copying the data: %w", err)
	}

	data, err := m.Storage.Get(ctx, source)
	if err != nil {
		metrics.UploadRejectionsCounter.WithLabelValues(metricUploadDataOperationLabel, "storage_read_failed").Inc()
		return fmt.Errorf("failed to obtain the data of the source remote secret when copying the data: %w", err)
	}

	auditLog := logs.AuditLog(ctx).WithValues("source-remote-secret", client.ObjectKeyFromObject(source), "target-remote-secret", client.ObjectKeyFromObject(rs))
	auditLog.Info("about to copy data from one remote secret to another")
	if err := m.Storage.Store(ctx, rs, data); err != nil {
		metrics.UploadRejectionsCounter.WithLabelValues(metricUploadDataOperationLabel, "storage_write_failed").Inc()
		auditLog.Error(err, "failed to copy data from one remote secret to another")
		return fmt.Errorf("failed to store the data copied from the source remote secret: %w", err)
	}
	auditLog.Info("successfully copied the data from source remote secret to target remote secret")

	rs.DataFrom = api.RemoteSecretDataFrom{}

	return nil
}

func (m *RemoteSecretMutator) checkHasPermissions(ctx context.Context, user authv1.UserInfo, sourceName, sourceNamespace string) error {
	sar := &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Name:      sourceName,
				Namespace: sourceNamespace,
				Verb:      "get",
				Group:     api.GroupVersion.Group,
				Version:   api.GroupVersion.Version,
				Resource:  "remotesecrets",
			},
			UID:    user.UID,
			User:   user.Username,
			Groups: user.Groups,
		},
	}

	if err := m.Client.Create(ctx, sar); err != nil {
		return fmt.Errorf("failed to create a subject access review to check if the user can copy data of remote secret: %w", err)
	}

	if !sar.Status.Allowed {
		if sar.Status.Reason != "" {
			return fmt.Errorf("%w: %s", errorCopyNotAllowed, sar.Status.Reason)
		} else {
			return errorCopyNotAllowed
		}
	}

	return nil
}
