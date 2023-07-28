/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/redhat-appstudio/remote-secret/pkg/commaseparated"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"

	"github.com/go-logr/logr"

	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var uploadSecretSelector = metav1.LabelSelector{
	MatchExpressions: []metav1.LabelSelectorRequirement{
		{
			Key:      api.UploadSecretLabel,
			Values:   []string{"remotesecret"},
			Operator: metav1.LabelSelectorOpIn,
		},
	},
}

var remoteSecretDoesntExist = errors.New("remote secret does not exist")

// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;watch;create;update;list;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;delete

// TokenUploadReconciler reconciles a Secret object
type TokenUploadReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	RemoteSecretStorage remotesecretstorage.RemoteSecretStorage
}

// SetupWithManager sets up the controller with the Manager.
func (r *TokenUploadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred, err := predicate.LabelSelectorPredicate(uploadSecretSelector)
	if err != nil {
		return fmt.Errorf("failed to construct the predicate for matching secrets. This should not happen: %w", err)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(pred)).
		Complete(r); err != nil {
		err = fmt.Errorf("failed to build the controller manager: %w", err)
		return err
	}
	return nil
}

func (r *TokenUploadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)
	lg.V(logs.DebugLevel).Info("starting reconciliation")
	defer logs.TimeTrackWithLazyLogger(func() logr.Logger { return lg }, time.Now(), "Reconcile Upload Secret")

	uploadSecret := &corev1.Secret{}

	if err := r.Get(ctx, req.NamespacedName, uploadSecret); err != nil {
		if kuberrors.IsNotFound(err) {
			lg.V(logs.DebugLevel).Info("upload secret already gone from the cluster. skipping reconciliation")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to load the upload secret from the cluster: %w", err)
	}

	if uploadSecret.DeletionTimestamp != nil {
		lg.V(logs.DebugLevel).Info("upload secret is being deleted. skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// We first find/create the RemoteSecret since we need it to store the data. Only after we have stored the data
	// in secretStorage can we delete the uploadSecret. The deletion triggers RS reconciliation in which the data is
	// fetched from the storage and propagated to the targets by RS controller.
	err := r.reconcileRemoteSecret(ctx, uploadSecret)

	// we immediately delete the Secret
	delErr := r.Delete(ctx, uploadSecret)
	if delErr != nil {
		// We failed to delete the secret, so we error out so that we can try again in the next reconciliation round.
		// We therefore also DON'T create the error event in this case like we do later on in this method.
		return ctrl.Result{}, fmt.Errorf("cannot delete the Secret: %w", err)
	}

	// NOTE: it is useless to return any error to the controller runtime after this point, because we just
	// deleted the secret that is being reconciled and so its repeated reconciliation would short-circuit
	// on the non-nil DeletionTimestamp. Therefore, in case of errors, we just create the error event and
	// return a "success" to the controller runtime.

	if err != nil {
		r.createErrorEvent(ctx, uploadSecret, err, lg)
	} else {
		r.tryDeleteEvent(ctx, uploadSecret.Name, req.Namespace, lg)
	}

	return ctrl.Result{}, nil
}

func determineActions(annos map[string]string) (createOrUpdate bool, partialUpdate bool, partialDelete bool) {
	if _, partialUpdate = annos[api.RemoteSecretPartialUpdateAnnotation]; partialUpdate {
		_, partialDelete = annos[api.RemoteSecretDeletedKeysAnnotation]
		// if we declare partial update, the createOrUpdate can never be true
		return
	}

	createOrUpdate = true
	return
}

func getKeysToDelete(annos map[string]string) []string {
	return commaseparated.Value(annos[api.RemoteSecretDeletedKeysAnnotation]).Values()
}

func (r *TokenUploadReconciler) reconcileRemoteSecret(ctx context.Context, uploadSecret *corev1.Secret) error {
	createOrUpdate, partialUpdate, partialDelete := determineActions(uploadSecret.Annotations)

	remoteSecret, err := r.findRemoteSecret(ctx, uploadSecret)
	if err != nil {
		return fmt.Errorf("attempt to find the remote secret failed: %w", err)
	}
	if createOrUpdate && remoteSecret == nil {
		remoteSecret, err = r.createRemoteSecret(ctx, uploadSecret)
		if err != nil {
			return fmt.Errorf("failed to create the remote secret: %w", err)
		}
	}

	if remoteSecret == nil {
		return remoteSecretDoesntExist
	}

	err = remoteSecret.ValidateUploadSecretType(uploadSecret)
	if err != nil {
		return fmt.Errorf("validation of upload secret failed: %w ", err)
	}

	auditLog := logs.AuditLog(ctx).WithValues("remoteSecret", client.ObjectKeyFromObject(remoteSecret))

	if createOrUpdate {
		auditLog.Info("manual secret upload initiated", "action", "UPDATE")
		err = r.RemoteSecretStorage.Store(ctx, remoteSecret, (*remotesecretstorage.SecretData)(&uploadSecret.Data))
		if err != nil {
			err = fmt.Errorf("failed to store the remote secret data: %w", err)
			auditLog.Error(err, "manual secret upload failed")
			return err
		}
		auditLog.Info("manual secret upload completed")
	}

	if partialUpdate {
		auditLog.Info("manual secret partial update initiated", "action", "READ")
		data, err := r.RemoteSecretStorage.Get(ctx, remoteSecret)
		if err != nil {
			err = fmt.Errorf("failed to read the secret data: %w", err)
			auditLog.Error(err, "manual secret partial update failed")
			return err
		}

		for key, val := range uploadSecret.Data {
			(*data)[key] = val
		}

		if partialDelete {
			for _, key := range getKeysToDelete(uploadSecret.Annotations) {
				delete(*data, key)
			}
		}
		auditLog.Info("manual secret partial update", "action", "UPDATE")
		err = r.RemoteSecretStorage.Store(ctx, remoteSecret, data)
		if err != nil {
			err = fmt.Errorf("failed to store the secret data: %w", err)
			auditLog.Error(err, "manual secret partial update failed")
			return err
		}
	}

	return nil
}

// reports error in both Log and Event in current Namespace
func (r *TokenUploadReconciler) createErrorEvent(ctx context.Context, secret *corev1.Secret, err error, lg logr.Logger) {
	r.tryDeleteEvent(ctx, secret.Name, secret.Namespace, lg)

	secretErrEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
		Message:        err.Error(),
		Reason:         "cannot process upload secret",
		InvolvedObject: corev1.ObjectReference{Namespace: secret.Namespace, Name: secret.Name, Kind: secret.Kind, APIVersion: secret.APIVersion},
		Type:           "Error",
		LastTimestamp:  metav1.NewTime(time.Now()),
	}

	err = r.Create(ctx, secretErrEvent)
	if err != nil {
		lg.Error(err, "event creation failed for upload secret")
	}

}

// Contract: having exactly one event if upload failed and no events if uploaded.
// For this need to delete the event every attempt
func (r *TokenUploadReconciler) tryDeleteEvent(ctx context.Context, secretName string, ns string, lg logr.Logger) {
	stored := &corev1.Event{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, stored)

	if err == nil {
		lg.V(logs.DebugLevel).Info("event found and will be deleted", "event.name", stored.Name)
		err = r.Delete(ctx, stored)
		if err != nil {
			lg.Error(err, "can not delete event")
		}
	}
}

func (r *TokenUploadReconciler) findRemoteSecret(ctx context.Context, uploadSecret *corev1.Secret) (*api.RemoteSecret, error) {
	lg := log.FromContext(ctx)

	remoteSecretName := uploadSecret.Annotations[api.RemoteSecretNameAnnotation]
	if remoteSecretName == "" {
		lg.V(logs.DebugLevel).Info("No remoteSecretName found, will try to create with generated ")
		return nil, nil
	}

	remoteSecret := api.RemoteSecret{}
	err := r.Get(ctx, types.NamespacedName{Name: remoteSecretName, Namespace: uploadSecret.Namespace}, &remoteSecret)

	if err != nil {
		if kuberrors.IsNotFound(err) {
			lg.V(logs.DebugLevel).Info("RemoteSecret NOT found, will try to create  ", "RemoteSecret.name", remoteSecret.Name)
			return nil, nil
		} else {
			return nil, fmt.Errorf("can not find RemoteSecret %s: %w ", remoteSecretName, err)
		}
	} else {
		lg.V(logs.DebugLevel).Info("RemoteSecret found : ", "RemoteSecret.name", remoteSecret.Name)
		return &remoteSecret, nil
	}
}

func (r *TokenUploadReconciler) createRemoteSecret(ctx context.Context, uploadSecret *corev1.Secret) (*api.RemoteSecret, error) {
	lg := log.FromContext(ctx)

	remoteSecret := api.RemoteSecret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: uploadSecret.Namespace,
		},
		Spec: api.RemoteSecretSpec{},
	}

	targetName, ok := uploadSecret.Annotations[api.TargetNamespaceAnnotation]
	if ok {
		targetSpec := api.RemoteSecretTarget{}
		targetSpec.Namespace = targetName
		remoteSecret.Spec.Targets = []api.RemoteSecretTarget{targetSpec}
	}

	remoteSecretName := uploadSecret.Annotations[api.RemoteSecretNameAnnotation]
	if remoteSecretName == "" {
		remoteSecret.GenerateName = "generated-"
	} else {
		remoteSecret.Name = remoteSecretName
	}

	err := r.Create(ctx, &remoteSecret)
	if err == nil {
		lg.V(logs.DebugLevel).Info("RemoteSecret created : ", "RemoteSecret.name", remoteSecret.Name, "targetName", targetName)
		return &remoteSecret, nil
	} else {
		return nil, fmt.Errorf("can not create RemoteSecret %s. Reason: %w", remoteSecret.Name, err)
	}
}
