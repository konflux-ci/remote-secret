package webhook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type RemoteSecretMutator struct {
	Storage secretstorage.SecretStorage
}

// +kubebuilder:webhook:path=/mutate-appstudio-redhat-com-v1beta1-remotesecret,mutating=true,failurePolicy=fail,sideEffects=None,groups=appstudio.redhat.com,resources=remotesecrets,verbs=create;update,versions=v1beta1,name=mremotesecret.kb.io,admissionReviewVersions=v1
var _ webhook.CustomDefaulter = &RemoteSecretMutator{}

func (a *RemoteSecretMutator) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	log.Info("Object", "obj", obj)

	rs, ok := obj.(*v1beta1.RemoteSecret)
	if !ok {
		return fmt.Errorf("expected a RemoteSecret but got a %T", obj)
	}

	secretData := rs.UploadData

	if len(secretData) != 0 {
		log.Info("Data DETECTED, upload it and delete from here", "Data", secretData)
		bytes, err := json.Marshal(secretData)
		if err != nil {
			return fmt.Errorf("failed to serialize data: %w", err)
		}
		uid := secretstorage.SecretID{
			Name:      rs.Name,
			Namespace: rs.Namespace,
		}

		err = a.Storage.Store(ctx, uid, bytes)
		if err != nil {
			return fmt.Errorf("Storage error %s", err)
		}

		//////////////////
		newData, err := a.Storage.Get(ctx, uid)
		if err != nil {
			return fmt.Errorf("Storage error (get) %s", err)
		}

		err = json.Unmarshal(newData, &secretData)
		log.Info("Data STORED ", "data", secretData)
		/////////////////

		// clean upload data
		rs.UploadData = map[string]string{}
		//[]v1beta1.KeyValue{}
	}

	return nil
}
