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

package controllers

import (
	"context"
	stdErrors "errors"
	"fmt"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/redhat-appstudio/remote-secret/pkg/rerror"

	"github.com/go-logr/logr"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/controllers/namespacetarget"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecrets"
	"github.com/redhat-appstudio/remote-secret/controllers/remotesecretstorage"
	opconfig "github.com/redhat-appstudio/remote-secret/pkg/config"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"github.com/redhat-appstudio/remote-secret/pkg/secretstorage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/finalizer"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	unexpectedObjectTypeError = stdErrors.New("unexpected object type")
)

const linkedObjectsFinalizerName = "appstudio.redhat.com/linked-objects"

type RemoteSecretReconciler struct {
	client.Client
	TargetClientFactory bindings.ClientFactory
	Scheme              *runtime.Scheme
	Configuration       *opconfig.OperatorConfiguration
	RemoteSecretStorage remotesecretstorage.RemoteSecretStorage
	finalizers          finalizer.Finalizers
}

//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=remotesecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=remotesecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=appstudio.redhat.com,resources=remotesecrets/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;watch;create;update;list;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts/token,verbs=create

var _ reconcile.Reconciler = (*RemoteSecretReconciler)(nil)

const storageFinalizerName = "appstudio.redhat.com/secret-storage" //#nosec G101 -- false positive, we're not storing any sensitive data using this

func (r *RemoteSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.finalizers = finalizer.NewFinalizers()
	if err := r.finalizers.Register(storageFinalizerName, &remoteSecretStorageFinalizer{storage: r.RemoteSecretStorage}); err != nil {
		return fmt.Errorf("failed to register the remote secret storage finalizer: %w", err)
	}
	if err := r.finalizers.Register(linkedObjectsFinalizerName, &remoteSecretLinksFinalizer{clientFactory: r.TargetClientFactory, storage: r.RemoteSecretStorage}); err != nil {
		return fmt.Errorf("failed to register the remote secret links finalizer: %w", err)
	}

	pred, err := predicate.LabelSelectorPredicate(uploadSecretSelector)
	if err != nil {
		return fmt.Errorf("failed to construct the predicate for matching secrets. This should not happen: %w", err)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&api.RemoteSecret{}).
		Watches(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			return linksToReconcileRequests(mgr.GetLogger(), mgr.GetScheme(), o)
		})).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.findRemoteSecretForUploadSecret),
			builder.WithPredicates(pred, predicate.Funcs{
				DeleteFunc: func(de event.DeleteEvent) bool { return true },
			}),
		).
		Watches(&source.Kind{Type: &corev1.ServiceAccount{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			return linksToReconcileRequests(mgr.GetLogger(), mgr.GetScheme(), o)
		})).
		Watches(&source.Kind{Type: &corev1.ServiceAccount{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			return r.findRemoteSecretsInNamespaceForAuthSA(o)
		}), builder.OnlyMetadata).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to configure the reconciler: %w", err)
	}
	return nil
}

func (r *RemoteSecretReconciler) findRemoteSecretForUploadSecret(secret client.Object) []reconcile.Request {
	remoteSecretName := secret.GetAnnotations()[api.RemoteSecretNameAnnotation]
	if remoteSecretName == "" {
		return []reconcile.Request{}
	}

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Namespace: secret.GetNamespace(),
				Name:      remoteSecretName,
			},
		},
	}
}

func linksToReconcileRequests(lg logr.Logger, scheme *runtime.Scheme, o client.Object) []reconcile.Request {
	ctx := context.Background()
	nsMarker := namespacetarget.NamespaceObjectMarker{}

	refs, err := nsMarker.GetReferencingTargets(ctx, o)
	if err != nil {
		var gvk schema.GroupVersionKind
		gvks, _, _ := scheme.ObjectKinds(o)
		if len(gvks) > 0 {
			gvk = gvks[0]
		}
		lg.Error(err, "failed to list the referencing targets of the object", "objectKey", client.ObjectKeyFromObject(o), "gvk", gvk)
	}

	if len(refs) == 0 {
		return nil
	}

	ret := make([]reconcile.Request, len(refs))

	for i, r := range refs {
		ret[i] = reconcile.Request{NamespacedName: r}
	}
	return ret
}

func (r *RemoteSecretReconciler) findRemoteSecretsInNamespaceForAuthSA(o client.Object) []reconcile.Request {
	ctx := context.Background()

	if _, ok := o.GetLabels()[bindings.RemoteSecretAuthServiceAccountLabel]; !ok {
		return nil
	}

	lg := log.FromContext(ctx)

	// get all the remote secrets from the current namespace
	list := api.RemoteSecretList{}
	if err := r.Client.List(ctx, &list, client.InNamespace(o.GetNamespace())); err != nil {
		lg.Error(err, "failed to list the remote secrets in a namespace while processing a change in authenticating service account")
		return nil
	}

	ret := make([]reconcile.Request, len(list.Items))

	for i := range list.Items {
		ret[i] = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])}
	}
	return ret
}

// Reconcile implements reconcile.Reconciler
func (r *RemoteSecretReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	lg := log.FromContext(ctx)
	lg.V(logs.DebugLevel).Info("starting reconciliation")
	defer logs.TimeTrackWithLazyLogger(func() logr.Logger { return lg }, time.Now(), "Reconcile RemoteSecret")

	remoteSecret := &api.RemoteSecret{}

	if err := r.Get(ctx, req.NamespacedName, remoteSecret); err != nil {
		if errors.IsNotFound(err) {
			lg.V(logs.DebugLevel).Info("RemoteSecret already gone from the cluster. skipping reconciliation")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get the RemoteSecret: %w", err)
	}

	finalizationResult, err := r.finalizers.Finalize(ctx, remoteSecret)
	if err != nil {
		// if the finalization fails, the finalizer stays in place, and so we don't want any repeated attempts until
		// we get another reconciliation due to cluster state change
		return ctrl.Result{Requeue: false}, fmt.Errorf("failed to finalize: %w", err)
	}
	if finalizationResult.Updated {
		lg.V(logs.DebugLevel).Info("finalizer wants to update the spec. updating it.")
		if err = r.Client.Update(ctx, remoteSecret); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update based on finalization result: %w", err)
		}
	}
	if finalizationResult.StatusUpdated {
		lg.V(logs.DebugLevel).Info("finalizer wants to update the status. updating it.")
		if err = r.Client.Status().Update(ctx, remoteSecret); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update the status based on finalization result: %w", err)
		}
	}

	if remoteSecret.DeletionTimestamp != nil {
		lg.V(logs.DebugLevel).Info("RemoteSecret is being deleted. skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// the reconciliation happens in stages, results of which are described in the status conditions.

	dataResult, err := handleStage(ctx, r.Client, remoteSecret, r.obtainData(ctx, remoteSecret))
	if err != nil || dataResult.Cancellation.Cancel {
		return dataResult.Cancellation.Result, err
	}

	deployResult, err := handleStage(ctx, r.Client, remoteSecret, r.deploy(ctx, remoteSecret, dataResult.ReturnValue))
	if err != nil || deployResult.Cancellation.Cancel {
		return deployResult.Cancellation.Result, err
	}

	return ctrl.Result{}, nil
}

// stageResult describes the result of reconciliation stage.
type stageResult[R any] struct {
	// Name is the name of the stage used in error reporting
	Name string
	// Condition is the condition describing the result of the stage in the remote secret's status.
	Condition metav1.Condition
	// ReturnValue is the result of this stage. It can be used by later stages.
	ReturnValue R
	// Cancellation describes whether and how to cancel the current reconciliation early, right after the stage.
	Cancellation cancellation
}

type cancellation struct {
	// Result contains the result to return when cancelling the current reconciliation.
	Result ctrl.Result
	// Cancel makes the current reconciliation stop early, right after this stage with the Result.
	Cancel bool
	// ReturnError is the error that will be returned from the reconciliation method if this stage is cancelling the reconciliation.
	ReturnError error
}

// handleStage tries to update the status with the condition from the provided result and returns error if the update failed or the stage itself failed before.
func handleStage[T any](ctx context.Context, cl client.Client, remoteSecret *api.RemoteSecret, result stageResult[T]) (stageResult[T], error) {
	meta.SetStatusCondition(&remoteSecret.Status.Conditions, result.Condition)

	if serr := cl.Status().Update(ctx, remoteSecret); serr != nil {
		return result, fmt.Errorf("failed to persist the stage result condition in the status after the stage %s: %w", result.Name, serr)
	}

	if result.Cancellation.Cancel || result.Cancellation.ReturnError != nil {
		return result, result.Cancellation.ReturnError
	} else {
		return result, nil
	}
}

// obtainData tries to find the data of the remote secret in the backing storage.
func (r *RemoteSecretReconciler) obtainData(ctx context.Context, remoteSecret *api.RemoteSecret) stageResult[*remotesecretstorage.SecretData] {
	result := stageResult[*remotesecretstorage.SecretData]{
		Name: "data-fetch",
	}

	secretData, err := r.RemoteSecretStorage.Get(ctx, remoteSecret)
	if err != nil {
		if stdErrors.Is(err, secretstorage.NotFoundError) {
			result.Condition = metav1.Condition{
				Type:    string(api.RemoteSecretConditionTypeDataObtained),
				Status:  metav1.ConditionFalse,
				Reason:  string(api.RemoteSecretReasonAwaitingTokenData),
				Message: "The data of the remote secret not found in storage. Please provide it.",
			}
			// we don't want to retry the reconciliation in this case, because the data is simply not present in the storage.
			// we will get notified once it appears there.
		} else {
			result.Condition = metav1.Condition{
				Type:    string(api.RemoteSecretConditionTypeDataObtained),
				Status:  metav1.ConditionFalse,
				Reason:  string(api.RemoteSecretReasonError),
				Message: err.Error(),
			}
			// we want to retry the reconciliation in this case because something else failed while we tried to get the data. so let's return the error
			result.Cancellation.ReturnError = err
		}
		// regardless of whether we want to repeat the reconciliation, we don't want to continue with the current one, because the remote secret
		// doesn't have any data to put into the target secrets.
		result.Cancellation.Cancel = true
		return result
	}

	result.Condition = metav1.Condition{
		Type:   string(api.RemoteSecretConditionTypeDataObtained),
		Status: metav1.ConditionTrue,
		Reason: string(api.RemoteSecretReasonDataFound),
	}

	result.ReturnValue = secretData

	return result
}

// deploy tries to deploy the secret to all the specified targets. It accumulates all errors, rather than stopping on the first one, so that we deploy
// to as many targets as possible.
func (r *RemoteSecretReconciler) deploy(ctx context.Context, remoteSecret *api.RemoteSecret, data *remotesecretstorage.SecretData) stageResult[any] {
	result := stageResult[any]{
		Name: "secret-deployment",
	}

	aerr := &rerror.AggregatedError{}
	r.processTargets(ctx, remoteSecret, data, aerr)

	var deploymentStatus metav1.ConditionStatus
	var deploymentReason api.RemoteSecretReason
	var deploymentMessage string

	if aerr.HasErrors() {
		log.FromContext(ctx).Error(aerr, "failed to deploy the secret to some targets")

		deploymentReason = api.RemoteSecretReasonPartiallyInjected
		deploymentStatus = metav1.ConditionFalse
		deploymentMessage = aerr.Error()
		// we want to retry the reconciliation because we failed to deploy to some targets in a retryable way
		result.Cancellation.Cancel = true
		result.Cancellation.ReturnError = aerr
	} else {
		// we might have no hard errors bubbling up but the individual targets might still have failed
		// in a way that is not retryable. let's check for that...
		hasErrors := false
		for _, ts := range remoteSecret.Status.Targets {
			if ts.Error != "" {
				hasErrors = true
			}
		}

		if hasErrors {
			deploymentReason = api.RemoteSecretReasonPartiallyInjected
			deploymentStatus = metav1.ConditionFalse
			deploymentMessage = "some targets were not successfully deployed"
		} else {
			deploymentReason = api.RemoteSecretReasonInjected
			deploymentStatus = metav1.ConditionTrue
		}
	}

	result.Condition = metav1.Condition{
		Type:    string(api.RemoteSecretConditionTypeDeployed),
		Status:  deploymentStatus,
		Reason:  string(deploymentReason),
		Message: deploymentMessage,
	}

	return result
}

// processTargets uses remotesecrets.ClassifyTargetNamespaces to find out what to do with targets in the remote secret spec and status
// and does what the classification tells it to.
func (r *RemoteSecretReconciler) processTargets(ctx context.Context, remoteSecret *api.RemoteSecret, secretData *remotesecretstorage.SecretData, errorAggregate *rerror.AggregatedError) {
	namespaceClassification := remotesecrets.ClassifyTargetNamespaces(remoteSecret)
	log.FromContext(ctx).V(logs.DebugLevel).Info("namespace classification", "classification", namespaceClassification)
	for specIdx, statusIdx := range namespaceClassification.Sync {
		spec := &remoteSecret.Spec.Targets[specIdx]
		var status *api.TargetStatus
		if statusIdx == -1 {
			// as per docs, ClassifyTargetNamespaces uses -1 to indicate that the target is not in the status.
			// So we just add a new empty entry to status and use that to deploy to the namespace.
			// deployToNamespace will fill it in.
			remoteSecret.Status.Targets = append(remoteSecret.Status.Targets, api.TargetStatus{})
			status = &remoteSecret.Status.Targets[len(remoteSecret.Status.Targets)-1]
		} else {
			status = &remoteSecret.Status.Targets[statusIdx]
		}
		err := r.deployToNamespace(ctx, remoteSecret, spec, status, secretData)
		if err != nil {
			errorAggregate.Add(err)
		}
	}

	for _, statusIndex := range namespaceClassification.Remove {
		err := r.deleteFromNamespace(ctx, remoteSecret, statusIndex)
		if err != nil {
			errorAggregate.Add(err)
		}
	}

	// mark the duplicates...
	for originalIdx, duplicates := range namespaceClassification.DuplicateTargetSpecs {
		for specIdx, statusIdx := range duplicates {
			var status *api.TargetStatus
			if statusIdx == -1 {
				remoteSecret.Status.Targets = append(remoteSecret.Status.Targets, api.TargetStatus{})
				status = &remoteSecret.Status.Targets[len(remoteSecret.Status.Targets)-1]
			} else {
				status = &remoteSecret.Status.Targets[statusIdx]
			}
			// clear out the status and just set the key and error
			*status = api.TargetStatus{
				ApiUrl:    remoteSecret.Spec.Targets[specIdx].ApiUrl,
				Namespace: remoteSecret.Spec.Targets[specIdx].Namespace,
				Error:     fmt.Sprintf("the target at the index %d is a duplicate of the target at the index %d", specIdx, originalIdx),
			}
		}
	}

	// and finally, remove the orphaned and deleted targets from the status
	toRemove := make([]remotesecrets.StatusTargetIndex, 0, len(namespaceClassification.Remove)+len(namespaceClassification.OrphanDuplicateStatuses))
	toRemove = append(toRemove, namespaceClassification.Remove...)
	toRemove = append(toRemove, namespaceClassification.OrphanDuplicateStatuses...)
	// sort the array in reverse order so that we can remove from the status without reindexing
	sort.Slice(toRemove, func(i, j int) bool {
		return toRemove[i] > toRemove[j]
	})

	for _, stIdx := range toRemove {
		remoteSecret.Status.Targets = append(remoteSecret.Status.Targets[:stIdx], remoteSecret.Status.Targets[stIdx+1:]...)
	}
}

// deployToNamespace deploys the secret to the provided tartet and fills in the provided status with the result of the deployment. The status will also contain the error
// if the deployment failed. This returns an error if the deployment fails (this is recorded in the target status) OR if the update of the status in k8s fails (this is,
// obviously, not recorded in the target status).
func (r *RemoteSecretReconciler) deployToNamespace(ctx context.Context, remoteSecret *api.RemoteSecret, targetSpec *api.RemoteSecretTarget, targetStatus *api.TargetStatus, data *remotesecretstorage.SecretData) error {
	debugLog := log.FromContext(ctx).V(logs.DebugLevel)

	var depErr, checkPointErr, syncErr, updateErr error

	depHandler, depErr := newDependentsHandler(ctx, r.TargetClientFactory, r.RemoteSecretStorage, remoteSecret, targetSpec, targetStatus)
	if depErr != nil {
		debugLog.Error(depErr, "failed to construct the dependents handler")
	}

	var checkPoint bindings.CheckPoint
	if depHandler != nil {
		checkPoint, checkPointErr = depHandler.CheckPoint(ctx)
		if checkPointErr != nil {
			debugLog.Error(checkPointErr, "failed to construct a checkpoint to rollback to in case of target deployment error")
		}
	}

	var deps *bindings.Dependents

	if depHandler != nil && checkPointErr == nil {
		deps, _, syncErr = depHandler.Sync(ctx, remoteSecret)
	}

	targetStatus.ApiUrl = targetSpec.ApiUrl
	targetStatus.ClusterCredentialsSecret = targetSpec.ClusterCredentialsSecret

	inconsistent := false

	if deps != nil {
		targetStatus.Namespace = deps.Secret.Namespace
		targetStatus.SecretName = deps.Secret.Name

		targetStatus.ServiceAccountNames = make([]string, len(deps.ServiceAccounts))
		for i, sa := range deps.ServiceAccounts {
			targetStatus.ServiceAccountNames[i] = sa.Name
		}
		targetStatus.Error = ""
	} else {
		targetStatus.Namespace = targetSpec.Namespace
		targetStatus.SecretName = ""
		targetStatus.ServiceAccountNames = []string{}
		// finalizer depends on this being non-empty only in situations where we never deployed anything to the
		// target.
		targetStatus.Error = rerror.AggregateNonNilErrors(depErr, checkPointErr, syncErr).Error()
		if stdErrors.Is(syncErr, bindings.DependentsInconsistencyError) {
			inconsistent = true
		}
	}

	updateErr = r.Client.Status().Update(ctx, remoteSecret)
	if syncErr != nil || updateErr != nil {
		if syncErr != nil {
			if inconsistent {
				debugLog.Info("encountered an inconsistency error", "error", syncErr.Error())
			} else {
				debugLog.Error(syncErr, "failed to sync the dependent objects")
			}
		}

		if updateErr != nil {
			debugLog.Error(updateErr, "failed to update the status with the info about dependent objects")
		}

		if rerr := depHandler.RevertTo(ctx, checkPoint); rerr != nil {
			debugLog.Error(rerr, "failed to revert the sync of the dependent objects of the remote secret after a failure", "statusUpdateError", updateErr, "syncError", syncErr)
		}
	} else if debugLog.Enabled() && depErr == nil && checkPointErr == nil {
		saks := make([]client.ObjectKey, len(deps.ServiceAccounts))
		for i, sa := range deps.ServiceAccounts {
			saks[i] = client.ObjectKeyFromObject(sa)
		}
		debugLog.Info("successfully synced dependent objects of remote secret", "remoteSecret", client.ObjectKeyFromObject(remoteSecret), "syncedSecret", client.ObjectKeyFromObject(deps.Secret))
	}

	// we want the inconsistency errors to be noted by the user, but we don't want them to
	// bubble up and cause reconcile retries
	if inconsistent {
		syncErr = nil
	}
	err := rerror.AggregateNonNilErrors(syncErr, updateErr)
	if err != nil {
		err = fmt.Errorf("failed to deploy to the namespace %s: %w", targetSpec.Namespace, err)
	}

	return err
}

func (r *RemoteSecretReconciler) deleteFromNamespace(ctx context.Context, remoteSecret *api.RemoteSecret, statusTargetIndex remotesecrets.StatusTargetIndex) error {
	dep, err := newDependentsHandler(ctx, r.TargetClientFactory, r.RemoteSecretStorage, remoteSecret, nil, &remoteSecret.Status.Targets[statusTargetIndex])
	if err != nil {
		return fmt.Errorf("failed to construct the handler to use for target cleanup: %w", err)
	}

	if err = dep.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to clean up dependent objects: %w", err)
	}

	// unlike in deployToNamespace, we DO NOT update the status here straight away. That is because doing that would mess up the indices
	// in the naming classification in processTargets which this method is a helper of.
	// It is safe to do so, because dep.Cleanup() above doesn't fail with missing objects, so if we get a failure halfway through removing
	// the secrets, we end up with inconsistent status, but that we will eventually solve itself when the reconciliation (which will be repeated
	// in that case) finally goes through completely.

	return nil
}

func newDependentsHandler(ctx context.Context, cf bindings.ClientFactory, st remotesecretstorage.RemoteSecretStorage, remoteSecret *api.RemoteSecret, targetSpec *api.RemoteSecretTarget, targetStatus *api.TargetStatus) (*bindings.DependentsHandler[*api.RemoteSecret], error) {
	cl, err := cf.GetClient(ctx, remoteSecret.Namespace, targetSpec, targetStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to construct a client to use for deploying to target: %w", err)
	}

	return &bindings.DependentsHandler[*api.RemoteSecret]{
		Target: &namespacetarget.NamespaceTarget{
			Client:       cl,
			TargetKey:    client.ObjectKeyFromObject(remoteSecret),
			SecretSpec:   &remoteSecret.Spec.Secret,
			TargetSpec:   targetSpec,
			TargetStatus: targetStatus,
		},
		SecretDataGetter: &remotesecrets.SecretDataGetter{
			Storage: st,
		},
		ObjectMarker: &namespacetarget.NamespaceObjectMarker{},
	}, nil
}

type remoteSecretStorageFinalizer struct {
	storage remotesecretstorage.RemoteSecretStorage
}

var _ finalizer.Finalizer = (*remoteSecretStorageFinalizer)(nil)

func (f *remoteSecretStorageFinalizer) Finalize(ctx context.Context, obj client.Object) (finalizer.Result, error) {
	err := f.storage.Delete(ctx, obj.(*api.RemoteSecret))
	if err != nil {
		err = fmt.Errorf("failed to delete the linked token during finalization of %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}
	return finalizer.Result{}, err
}

type remoteSecretLinksFinalizer struct {
	clientFactory bindings.ClientFactory
	storage       remotesecretstorage.RemoteSecretStorage
}

//var _ finalizer.Finalizer = (*linkedObjectsFinalizer)(nil)

// Finalize removes the secret and possibly also service account synced to the actual binging being deleted
func (f *remoteSecretLinksFinalizer) Finalize(ctx context.Context, obj client.Object) (finalizer.Result, error) {
	res := finalizer.Result{}
	remoteSecret, ok := obj.(*api.RemoteSecret)
	if !ok {
		return res, unexpectedObjectTypeError
	}

	lg := log.FromContext(ctx).V(logs.DebugLevel)

	key := client.ObjectKeyFromObject(remoteSecret)

	lg.Info("linked objects finalizer starting to clean up dependent objects", "remoteSecret", key)

	for i := range remoteSecret.Status.Targets {
		ts := remoteSecret.Status.Targets[i]
		// the error is set in the deployToNamespace function and is non-empty if we were unable to even
		// start deploying or if the deployment failed.
		// We never set this in the deleteFromNamespace function, which means that a failure to clean up
		// a target will still force another clean up attempt here.
		//
		// Therefore, it is safe to just skip this target, if it has a non-empty error. We can be sure
		// nothing exists in the target namespace.
		if ts.Error != "" {
			continue
		}
		dep, err := newDependentsHandler(ctx, f.clientFactory, f.storage, remoteSecret, nil, &ts)
		if err != nil {
			// we're in the finalizer and we failed to even construct the dependents handler.
			lg.Error(err, "failed to construct the dependents handler to clean up the target in the finalizer")
			return res, fmt.Errorf("failed to construct the dependents handler to clean up the target in the finalizer: %w", err)
		}
		if err := dep.Cleanup(ctx); err != nil {
			lg.Error(err, "failed to clean up the dependent objects in the finalizer", "binding", client.ObjectKeyFromObject(remoteSecret))
			return res, fmt.Errorf("failed to clean up dependent objects in the finalizer: %w", err)
		}
	}

	lg.Info("linked objects finalizer completed without failure", "binding", key)

	return res, nil
}
