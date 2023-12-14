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

package namespacetarget

import (
	"context"
	"errors"
	"fmt"
	"strings"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"github.com/redhat-appstudio/remote-secret/pkg/commaseparated"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceObjectMarker struct {
}

var malformedManagingAnnotation = errors.New("the ManagingRemoteSecret Annotation is malformed, this should not happen")

var _ bindings.ObjectMarker = (*NamespaceObjectMarker)(nil)

// IsManaged implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) IsManagedBy(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, error) {
	annos := obj.GetAnnotations()
	refed, _ := m.IsReferencedBy(ctx, rs, obj)
	return refed && annos[api.ManagingRemoteSecretNameAnnotation] == rs.String(), nil
}

func (m *NamespaceObjectMarker) IsManagedByOther(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, client.ObjectKey, error) {
	managingValue, managingPresent := obj.GetAnnotations()[api.ManagingRemoteSecretNameAnnotation]
	if !managingPresent {
		return false, client.ObjectKey{}, nil
	}
	if managingValue == rs.String() {
		return false, client.ObjectKey{}, nil
	}

	namespacedName := strings.Split(managingValue, "/")
	if len(namespacedName) != 2 {
		return false, client.ObjectKey{}, fmt.Errorf("%w: %s", malformedManagingAnnotation, managingValue)
	}
	return true, client.ObjectKey{Namespace: namespacedName[0], Name: namespacedName[1]}, nil
}

// IsReferenced implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) IsReferencedBy(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, error) {
	annos := obj.GetAnnotations()
	labels := obj.GetLabels()

	if labels[api.LinkedByRemoteSecretLabel] != "true" {
		return false, nil
	}

	return commaseparated.Value(annos[api.LinkedRemoteSecretsAnnotation]).Contains(rs.String()), nil
}

// ListManagedOptions implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) ListManagedOptions(ctx context.Context, rs client.ObjectKey) ([]client.ListOption, error) {
	return m.ListReferencedOptions(ctx, rs)
}

// ListReferencedOptions implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) ListReferencedOptions(ctx context.Context, rs client.ObjectKey) ([]client.ListOption, error) {
	return []client.ListOption{
		client.MatchingLabels{
			api.LinkedByRemoteSecretLabel: "true",
		},
	}, nil
}

// MarkManaged implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) MarkManaged(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, error) {
	refChanged, _ := m.MarkReferenced(ctx, rs, obj)

	value := rs.String()
	shouldChange := false
	annos := obj.GetAnnotations()

	if annos == nil {
		annos = map[string]string{}
		obj.SetAnnotations(annos)
		shouldChange = true
	} else {
		shouldChange = annos[api.ManagingRemoteSecretNameAnnotation] != value
	}

	if shouldChange {
		annos[api.ManagingRemoteSecretNameAnnotation] = value
	}

	return refChanged || shouldChange, nil
}

// MarkReferenced implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) MarkReferenced(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, error) {
	shouldChange := false
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
		obj.SetLabels(labels)
		shouldChange = true
	} else {
		shouldChange = labels[api.LinkedByRemoteSecretLabel] != "true"
	}

	labels[api.LinkedByRemoteSecretLabel] = "true"

	annos := obj.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
		obj.SetAnnotations(annos)
		shouldChange = true
	}

	link := rs.String()

	val := commaseparated.Value(annos[api.LinkedRemoteSecretsAnnotation])
	shouldChange = !val.Contains(link) || shouldChange

	if shouldChange {
		val.Add(link)
		annos[api.LinkedRemoteSecretsAnnotation] = val.String()
	}

	return shouldChange, nil
}

// UnmarkManaged implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) UnmarkManaged(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, error) {
	annos := obj.GetAnnotations()
	if annos == nil {
		return false, nil
	}

	val := annos[api.ManagingRemoteSecretNameAnnotation]

	if val == rs.String() {
		delete(annos, api.ManagingRemoteSecretNameAnnotation)
		return true, nil
	}

	return false, nil
}

// UnmarkReferenced implements bindings.ObjectMarker
func (m *NamespaceObjectMarker) UnmarkReferenced(ctx context.Context, rs client.ObjectKey, obj client.Object) (bool, error) {
	wasManaged, _ := m.UnmarkManaged(ctx, rs, obj)

	annos := obj.GetAnnotations()
	if annos == nil {
		return wasManaged, nil
	}

	link := rs.String()

	val := commaseparated.Value(annos[api.LinkedRemoteSecretsAnnotation])
	containsLink := val.Contains(link)

	if containsLink {
		val.Remove(link)
	}

	unlabeled := false
	if val.Len() == 0 {
		labels := obj.GetLabels()
		if labels != nil && labels[api.LinkedByRemoteSecretLabel] == "true" {
			delete(labels, api.LinkedByRemoteSecretLabel)
			unlabeled = true
		}
		delete(annos, api.LinkedRemoteSecretsAnnotation)
	} else {
		annos[api.LinkedRemoteSecretsAnnotation] = val.String()
	}

	return unlabeled || wasManaged || containsLink, nil
}

// GetReferencingTargets implements bindings.ObjectMarker
func (*NamespaceObjectMarker) GetReferencingTargets(ctx context.Context, obj client.Object) ([]types.NamespacedName, error) {
	val := commaseparated.Value(obj.GetAnnotations()[api.LinkedRemoteSecretsAnnotation])

	ret := make([]types.NamespacedName, val.Len())

	for i, v := range val.Values() {
		names := strings.Split(v, string(types.Separator))
		ret[i].Name = names[1]
		ret[i].Namespace = names[0]
	}

	return ret, nil
}
