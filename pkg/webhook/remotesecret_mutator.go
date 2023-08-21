package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type RemoteSecretMutator struct {
	Storage secretstorage.SecretStorage
}

// +kubebuilder:webhook:path=/mutate-appstudio-redhat-com-v1beta1-remotesecret,mutating=true,failurePolicy=fail,sideEffects=None,groups=appstudio.redhat.com,resources=remotesecrets,verbs=create;update,versions=v1beta1,name=mremotesecret.kb.io,admissionReviewVersions=v1
var _ webhook.CustomDefaulter = &RemoteSecretMutator{}

func (a *RemoteSecretMutator) Default(ctx context.Context, obj runtime.Object) error {
	rs, ok := obj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("%w: %T", errGotNonSecret, obj)
	}
	auditLog := logs.AuditLog(ctx).WithValues("remoteSecret", client.ObjectKeyFromObject(rs))

	secretData := rs.UploadData

	if len(secretData) != 0 {
		auditLog.Info("webhook data upload initiated")
		bytes, err := json.Marshal(secretData)
		if err != nil {
			err = fmt.Errorf("failed to serialize data: %w", err)
			auditLog.Error(err, "webhook data upload failed")
			return err
		}
		secretID := secretstorage.SecretID{
			Name:      rs.Name,
			Namespace: rs.Namespace,
		}

		err = a.Storage.Store(ctx, secretID, bytes)
		if err != nil {
			err = fmt.Errorf("storage error on data save: %w", err)
			auditLog.Error(err, "webhook data upload failed")
			return err
		}

		auditLog.Info("webhook data upload completed")

		// clean upload data
		rs.UploadData = map[string]string{}
		//[]v1beta1.KeyValue{}
	}

	return nil
}
