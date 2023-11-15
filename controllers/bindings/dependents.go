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
	"errors"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var DependentsInconsistencyError = errors.New("inconsistency detected when deploying dependent objects")

// DependentsHandler is taking care of the dependent objects of the provided target.
type DependentsHandler[K any] struct {
	Target           SecretDeploymentTarget
	SecretDataGetter SecretDataGetter[K]
	ObjectMarker     ObjectMarker
}

// Dependents represent the secret and the list of the service accounts that are
// linked to a deployment target of a dependents handler.
type Dependents struct {
	Secret          *corev1.Secret
	ServiceAccounts []*corev1.ServiceAccount
}

type serviceAccountLink struct {
	secret          bool
	imagePullSecret bool
}

type serviceAccountNamesAndLinkTypes map[string]serviceAccountLink

// CheckPoint is an opaque struct representing the state of the dependent objects at some point in time.
// It can be used in the DependentsHandler.RevertTo method to delete the secret/service accounts from the cluster
// that have been created after an instance of this struct has been returned from the DependentsHandler.CheckPoint
// method.
type CheckPoint struct {
	secretName          string
	serviceAccountNames serviceAccountNamesAndLinkTypes
}

// CheckPoint creates an instance of CheckPoint struct that captures the secret name and the list of known service account
// names from the deployment target associated with the DependentsHandler. This can later be used to revert back to that
// state again.
// See RevertTo for more details.
func (d *DependentsHandler[K]) CheckPoint(ctx context.Context) (CheckPoint, error) {
	secretName := d.Target.GetActualSecretName()
	names := make(map[string]serviceAccountLink, len(d.Target.GetActualServiceAccountNames()))
	for _, n := range d.Target.GetActualServiceAccountNames() {
		link, err := d.detectLinks(ctx, secretName, n)
		if err != nil {
			return CheckPoint{}, err
		}
		names[n] = link
	}

	return CheckPoint{
		secretName:          secretName,
		serviceAccountNames: names,
	}, nil
}

func (d *DependentsHandler[K]) detectLinks(ctx context.Context, secretName string, saName string) (serviceAccountLink, error) {
	link := serviceAccountLink{}

	sa := &corev1.ServiceAccount{}
	// we can ignore the error here, because this is essentially the same as the SA having no links. Sync will create it, if needed,
	// and RevertTo will ignore it, because it doesn't exist (it only works with the SAs actually present in the cluster).
	_ = d.Target.GetClient().Get(ctx, client.ObjectKey{Name: saName, Namespace: d.Target.GetTargetNamespace()}, sa)

	for _, s := range sa.Secrets {
		if s.Name == secretName {
			link.secret = true
			break
		}
	}

	for _, is := range sa.ImagePullSecrets {
		if is.Name == secretName {
			link.imagePullSecret = true
			break
		}
	}

	return link, nil
}

func (d *DependentsHandler[K]) Sync(ctx context.Context, dataKey K) (*Dependents, string, error) {
	// syncing the service accounts and secrets is a 3 step process.
	// First, an empty service account needs to be created.
	// Second, a secret linking to the service account needs to be created.
	// Third, the service account needs to be updated with the link to the secret.

	secretsHandler, saHandler := d.childHandlers()

	serviceAccounts, errorReason, err := saHandler.Sync(ctx)
	if err != nil {
		return nil, errorReason, err
	}

	staleSecret, err := secretsHandler.GetStale(ctx)
	if err != nil {
		return nil, string(ErrorReasonSecretUpdate), err
	}

	sec, errorReason, err := secretsHandler.Sync(ctx, dataKey, staleSecret != nil)
	if err != nil {
		return nil, errorReason, err
	}

	if err = saHandler.LinkToSecret(ctx, serviceAccounts, sec); err != nil {
		return nil, errorReason, err
	}

	if staleSecret != nil {
		for _, sa := range serviceAccounts {
			attempt := func() (client.Object, error) {
				updated := saHandler.Unlink(staleSecret, sa)
				if updated {
					return sa, nil
				} else {
					return nil, nil
				}
			}
			err := updateWithRetries(serviceAccountUpdateRetryCount, ctx, d.Target.GetClient(), attempt, "retrying unlinking of stale secret from SA due to a conflict",
				fmt.Sprintf("failed to update the service account '%s' with the link to the secret '%s' while processing the deployment target (%s) '%s'", sa.Name, staleSecret.Name, d.Target.GetType(), d.Target.GetTargetObjectKey()))
			if err != nil {
				return nil, string(ErrorReasonServiceAccountUpdate),
					fmt.Errorf("failed to unlink the stale secret %s from the service account %s while processing the deployment target (%s) %s: %w",
						client.ObjectKeyFromObject(staleSecret),
						client.ObjectKeyFromObject(sa),
						d.Target.GetType(),
						d.Target.GetTargetObjectKey(),
						err)
			}
		}

		// now that all the SAs are unlinked, we can finally delete the secret
		if err := d.Target.GetClient().Delete(ctx, staleSecret); err != nil {
			if !k8serrors.IsNotFound(err) {
				return nil, string(ErrorReasonSecretUpdate), fmt.Errorf("failed to delete the stale secret %s when processing the deployment target (%s) %s: %w",
					client.ObjectKeyFromObject(staleSecret),
					d.Target.GetType(),
					d.Target.GetTargetObjectKey(),
					err)
			}
		}
	}

	deps := &Dependents{
		Secret:          sec,
		ServiceAccounts: serviceAccounts,
	}

	return deps, "", nil
}

func (d *DependentsHandler[K]) Cleanup(ctx context.Context) error {
	secretsHandler, saHandler := d.childHandlers()

	sal, err := saHandler.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list the service accounts to clean up for the secret deployment target (%s) %s: %w",
			d.Target.GetType(),
			d.Target.GetTargetObjectKey(),
			err)
	}

	sl, err := secretsHandler.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list the secrets to clean for the secret deployment target (%s) %s: %w",
			d.Target.GetType(),
			d.Target.GetTargetObjectKey(),
			err)
	}

	for _, sa := range sal {
		if managed, err := d.ObjectMarker.IsManagedBy(ctx, d.Target.GetTargetObjectKey(), sa); err != nil {
			return fmt.Errorf("failed to determine if the service account (%s) is managed while processing the secret deployment target (%s) %s: %w",
				client.ObjectKeyFromObject(sa),
				d.Target.GetType(),
				d.Target.GetTargetObjectKey(),
				err)
		} else if managed {
			if err := d.Target.GetClient().Delete(ctx, sa); err != nil {
				if !k8serrors.IsNotFound(err) {
					return fmt.Errorf("failed to delete the managed service account %s while cleaning up dependent objects of the secret deployment target (%s) %s: %w",
						client.ObjectKeyFromObject(sa),
						d.Target.GetType(),
						d.Target.GetTargetObjectKey(),
						err)
				}
			}
		} else {
			persist := false
			for _, s := range sl {
				// Unlink must go first, because Go only has lazy bool eval
				persist = saHandler.Unlink(s, sa) || persist
			}
			if persist {
				if err := d.Target.GetClient().Update(ctx, sa); err != nil {
					return fmt.Errorf("failed to remove the linked secrets from the service account %s while cleaning up dependent objects of the secret deployment target (%s) %s: %w",
						client.ObjectKeyFromObject(sa),
						d.Target.GetType(),
						d.Target.GetTargetObjectKey(),
						err)
				}
			}
		}
	}

	for _, s := range sl {
		if err := d.Target.GetClient().Delete(ctx, s); err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete the secret %s while cleaning up dependent objects of secret deployment target (%s) %s: %w",
					client.ObjectKeyFromObject(s),
					d.Target.GetType(),
					d.Target.GetTargetObjectKey(),
					err)
			}
		}
	}

	return nil
}

// RevertTo reverts the reconciliation "transaction". I.e. this should be called after Sync in case the subsequent steps in the reconciliation
// fail and the operator needs to revert the changes made in sync so that the changes remain idempontent. The provided checkpoint represents
// the state obtained from the DependentsHandler.Target prior to making any changes by Sync().
// Note that currently this method is only able to delete secrets/service accounts that should not be in the cluster. It cannot "undelete"
// what has been deleted from the cluster. That should be OK though because we don't delete stuff during the Sync call.
func (d *DependentsHandler[K]) RevertTo(ctx context.Context, checkPoint CheckPoint) error {
	secretHandler, serviceAccountHandler := d.childHandlers()

	sl, err := secretHandler.List(ctx)
	if err != nil {
		return err
	}
	for _, s := range sl {
		if s.Name != checkPoint.secretName {
			if err := d.Target.GetClient().Delete(ctx, s); err != nil {
				if !k8serrors.IsNotFound(err) {
					return fmt.Errorf("failed to delete obsolete synced secret %s: %w", s.Name, err)
				}
			}
		}
	}

	sal, err := serviceAccountHandler.List(ctx)
	if err != nil {
		return err
	}

	for _, sa := range sal {
		attempt := func() (client.Object, error) {
			needsUpdate := false

			if saLink, ok := checkPoint.serviceAccountNames[sa.Name]; ok {
				// ok, the SA is in the list of the service accounts to revert to. The only thing we need to make sure is that
				// it links to the "old" secret name and doesn't link to the new secret name.
				// We also re-label it if needed.
				if checkPoint.secretName != d.Target.GetActualSecretName() {
					needsUpdate = serviceAccountHandler.unlinkSecretByName(d.Target.GetActualSecretName(), sa)
				}

				if saLink.secret {
					needsUpdate = serviceAccountHandler.linkSecretByName(sa, checkPoint.secretName, api.ServiceAccountLinkTypeSecret) || needsUpdate
				}

				if saLink.imagePullSecret {
					needsUpdate = serviceAccountHandler.linkSecretByName(sa, checkPoint.secretName, api.ServiceAccountLinkTypeImagePullSecret) || needsUpdate
				}
			} else {
				// so the SA is marked as linked to the target in the cluster, but it is not in the list of SAs in the state we are reverting to.
				// if the SA is managed, we can just delete it, but if it is not managed, we need to make sure the linking of secrets is accurate.

				if managed, err := d.ObjectMarker.IsManagedBy(ctx, d.Target.GetTargetObjectKey(), sa); err != nil {
					// let's retry, we just failed to check that labeling of the SA...
					return nil, fmt.Errorf("failed to determine whether the service account %s is managed while processing the deployment target (%s) %s: %w",
						client.ObjectKeyFromObject(sa),
						d.Target.GetType(),
						d.Target.GetTargetObjectKey(),
						err)
				} else if managed {
					if err := d.Target.GetClient().Delete(ctx, sa); err != nil {
						if !k8serrors.IsNotFound(err) {
							return nil, backoff.Permanent(fmt.Errorf("failed to delete obsolete service account %s originally linked to the secret deployment target (%s) %s: %w", //nolint:wrapcheck // this is just signalling to backoff.. will not bubble up.
								sa.Name,
								d.Target.GetType(),
								d.Target.GetTargetObjectKey(),
								err))
						}
					}
					// we don't need to do anything more on the SA because we just deleted it :)
					return nil, nil
				}

				serviceAccountHandler.unlinkSecretByName(d.Target.GetActualSecretName(), sa)
				serviceAccountHandler.unlinkSecretByName(checkPoint.secretName, sa)

				// the SA should not be linked to the target, so let's mark it as such...
				if _, err := d.ObjectMarker.UnmarkReferenced(ctx, d.Target.GetTargetObjectKey(), sa); err != nil {
					// let's retry, we just failed to unmark the SA as referenced (this does not actually persist anything, so we should be safe to retry).
					return nil, fmt.Errorf("failed to unmark the SA %s as referenced to the deployment target (%s) %s: %w",
						client.ObjectKeyFromObject(sa),
						d.Target.GetType(),
						d.Target.GetTargetObjectKey(),
						err)
				}

				needsUpdate = true
			}

			if needsUpdate {
				return sa, nil
			}

			return nil, nil
		}

		err = updateWithRetries(serviceAccountUpdateRetryCount, ctx, d.Target.GetClient(), attempt, "retry to update the SA while reverting to previous state of secret links", "failed to update the service account")

		if err != nil {
			return fmt.Errorf("failed to update the service account %s to revert it to prior state while recovering from failed secret deployment target (%s) %s reconciliation: %w",
				sa.Name,
				d.Target.GetType(),
				d.Target.GetTargetObjectKey(),
				err)
		}
	}

	return nil
}

// childHandlers is a utility function instantiating the auxilliary handlers for secrets and service accounts.
func (d *DependentsHandler[K]) childHandlers() (*secretHandler[K], *serviceAccountHandler) {
	secretsHandler := &secretHandler[K]{
		Target:           d.Target,
		ObjectMarker:     d.ObjectMarker,
		SecretDataGetter: d.SecretDataGetter,
	}

	saHandler := &serviceAccountHandler{
		Target:       d.Target,
		ObjectMarker: d.ObjectMarker,
	}

	return secretsHandler, saHandler
}
