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

package bindings

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/redhat-appstudio/remote-secret/pkg/logs"
	"github.com/redhat-appstudio/remote-secret/pkg/sync"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// pre-allocated empty map so that we don't have to allocate new empty instances in the serviceAccountSecretDiffOpts
	emptySecretData = map[string][]byte{}

	// secretIgnoredFields is TypeMeta + all most of the fields from ObjectMeta.
	// We only care for changes in name, generateName, namespace, finalizers, labels and annotations and therefore
	// should react to changes only in those.
	secretIgnoredFields = []string{
		"TypeMeta",
		"ObjectMeta.SelfLink",
		"ObjectMeta.UID",
		"ObjectMeta.ResourceVersion",
		"ObjectMeta.Generation",
		"ObjectMeta.CreationTimestamp",
		"ObjectMeta.DeletionTimestamp",
		"ObjectMeta.DeletionGracePeriodSeconds",
		"ObjectMeta.OwnerReferences",
		"ObjectMeta.ManagedFields",
	}

	secretDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(corev1.Secret{}, secretIgnoredFields...),
	}

	// the service account secrets are treated specially by Kubernetes that automatically adds "ca.crt", "namespace" and
	// "token" entries into the secret's data.
	serviceAccountSecretDiffOpts = cmp.Options{
		cmpopts.IgnoreFields(corev1.Secret{}, secretIgnoredFields...),
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.Last().String() == ".Data"
		}, cmp.Comparer(func(a map[string][]byte, b map[string][]byte) bool {
			// cmp.Equal short-circuits if it sees nil maps - but we don't want that...
			if a == nil {
				a = emptySecretData
			}
			if b == nil {
				b = emptySecretData
			}

			return cmp.Equal(a, b, cmpopts.IgnoreMapEntries(func(key string, _ []byte) bool {
				switch key {
				case "ca.crt", "namespace", "token":
					return true
				default:
					return false
				}
			}))
		}),
		),
	}
)

type secretHandler[K any] struct {
	Target           SecretDeploymentTarget
	ObjectMarker     ObjectMarker
	SecretDataGetter SecretDataGetter[K]
}

// GetStale detects whether the secret referenced by the target is stale and needs to be replaced by a new one.
// A secret in the target can become stale if it no longer corresponds to the spec of the target.
func (h *secretHandler[K]) GetStale(ctx context.Context) (*corev1.Secret, error) {
	existingSecretName := h.Target.GetActualSecretName()
	spec := h.Target.GetSpec()
	if existingSecretName == "" || NameCorresponds(existingSecretName, spec.Name, spec.GenerateName) {
		return nil, nil
	}

	existingSecret := &corev1.Secret{}
	err := h.Target.GetClient().Get(ctx, client.ObjectKey{Name: existingSecretName, Namespace: h.Target.GetTargetNamespace()}, existingSecret)
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return existingSecret, fmt.Errorf("failed to detect whether the secret is stale: %w", err)
	} else {
		return existingSecret, nil
	}
}

// Sync creates or updates the secret with the data from the given key. The recreate flag can be used to force the creation of a new secret even
// if the target already reports an existing secret using its GetActualSecretName method. This can be used to deal with the stale secrets (see GetStale method).
func (h *secretHandler[K]) Sync(ctx context.Context, key K, recreate bool) (*corev1.Secret, string, error) {
	data, errorReason, err := h.SecretDataGetter.GetData(ctx, key)
	if err != nil {
		return nil, errorReason, fmt.Errorf("failed to obtain the secret data: %w", err)
	}

	desiredSpec := h.Target.GetSpec()
	secretName := h.Target.GetActualSecretName()
	if recreate || secretName == "" {
		secretName = desiredSpec.Name
	}

	diffOpts := secretDiffOpts

	if desiredSpec.Type == corev1.SecretTypeServiceAccountToken {
		diffOpts = serviceAccountSecretDiffOpts
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:         secretName,
			GenerateName: desiredSpec.GenerateName,
			Namespace:    h.Target.GetTargetNamespace(),
			Labels:       desiredSpec.Labels,
			Annotations:  desiredSpec.Annotations,
		},
		Data: data,
		Type: desiredSpec.Type,
	}

	// we need to construct the list of labels that we want managed on the target secret
	// (i.e the list of keys that we want to delete from the maps if they are no longer desired
	// in our spec).
	// Maybe a little bit non-intuitively, that actually is the set of labels/annos already present
	// on the secret. We need those that are in the desiredSpec to stay and those that are not
	// there to disappear. So by declaring the already present labels as managed we can make that
	// happen.
	managedLabels := h.Target.GetActualManagedLabels()
	managedAnnos := h.Target.GetActualManagedAnnotations()

	if secret.GenerateName == "" {
		secret.GenerateName = h.Target.GetTargetObjectKey().Name + "-secret-"
	}

	_, err = h.ObjectMarker.MarkManaged(ctx, h.Target.GetTargetObjectKey(), secret)
	if err != nil {
		return nil, string(ErrorReasonSecretUpdate), fmt.Errorf("failed to mark the secret as managed in the deployment target (%s): %w", h.Target.GetType(), err)
	}

	syncer := sync.New(h.Target.GetClient())

	lg := log.FromContext(ctx).V(logs.DebugLevel)
	lg.Info("syncing binding secret", "secret", secret, "secretMetadata", &secret.ObjectMeta)

	_, obj, err := syncer.Sync(ctx, nil, secret, diffOpts, sync.LabelsAndAnnotationsSyncOptions{
		ManagedLabelKeys:      managedLabels,
		ManagedAnnotationKeys: managedAnnos,
	})
	if err != nil {
		return nil, string(ErrorReasonSecretUpdate), fmt.Errorf("failed to sync the secret with the token data: %w", err)
	}
	return obj.(*corev1.Secret), "", nil
}

func (h *secretHandler[K]) List(ctx context.Context) ([]*corev1.Secret, error) {
	sl := &corev1.SecretList{}
	opts, err := h.ObjectMarker.ListManagedOptions(ctx, h.Target.GetTargetObjectKey())
	if err != nil {
		return nil, fmt.Errorf("failed to formulate the options to list the secrets in the deployment target (%s): %w", h.Target.GetType(), err)
	}

	opts = append(opts, client.InNamespace(h.Target.GetTargetNamespace()))

	lg := log.FromContext(ctx).V(logs.DebugLevel)
	if err := h.Target.GetClient().List(ctx, sl, opts...); err != nil {
		return []*corev1.Secret{}, fmt.Errorf("failed to list the secrets associated with the deployment target (%s) %+v: %w", h.Target.GetType(), h.Target.GetTargetObjectKey(), err)
	}

	lg.Info("listing secrets managed by target", "targetType", h.Target.GetType(), "targetKey", h.Target.GetTargetObjectKey(), "targetNamespace", h.Target.GetTargetNamespace(), "opts", opts, "secretCount", len(sl.Items))

	ret := []*corev1.Secret{}
	for i := range sl.Items {
		if ok, err := h.ObjectMarker.IsManagedBy(ctx, h.Target.GetTargetObjectKey(), &sl.Items[i]); err != nil {
			return []*corev1.Secret{}, fmt.Errorf("failed to determine if the secret %s is managed while processing the deployment target (%s) %s: %w",
				client.ObjectKeyFromObject(&sl.Items[i]),
				h.Target.GetType(),
				h.Target.GetTargetObjectKey(),
				err)
		} else if ok {
			ret = append(ret, &sl.Items[i])
		}
	}

	return ret, nil
}
