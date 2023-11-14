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
	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/redhat-appstudio/remote-secret/controllers/bindings"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespaceTarget is the SecretDeploymentTarget that deploys the secrets and service accounts to some namespace on the cluster.
type NamespaceTarget struct {
	Client       client.Client
	TargetKey    client.ObjectKey
	SecretSpec   *api.LinkableSecretSpec
	TargetSpec   *api.RemoteSecretTarget
	TargetStatus *api.TargetStatus
}

var _ bindings.SecretDeploymentTarget = (*NamespaceTarget)(nil)

func (t *NamespaceTarget) GetActualManagedAnnotations() []string {
	ret := make([]string, 0, len(t.TargetStatus.Secret.Annotations))
	for k := range t.TargetStatus.Secret.Annotations {
		ret = append(ret, k)
	}
	return ret
}

func (t *NamespaceTarget) GetActualManagedLabels() []string {
	ret := make([]string, 0, len(t.TargetStatus.Secret.Labels))
	for k := range t.TargetStatus.Secret.Labels {
		ret = append(ret, k)
	}
	return ret
}

func (t *NamespaceTarget) GetSpec() api.LinkableSecretSpec {
	ret := t.SecretSpec
	if t.TargetSpec.Secret != nil {
		ret = t.SecretSpec.DeepCopy()
		if t.TargetSpec.Secret.Name != "" {
			ret.Name = t.TargetSpec.Secret.Name
		}
		if t.TargetSpec.Secret.GenerateName != "" {
			ret.GenerateName = t.TargetSpec.Secret.GenerateName
		}
		if t.TargetSpec.Secret.Labels != nil {
			ret.Labels = make(map[string]string, len(*t.TargetSpec.Secret.Labels))
			for k, v := range *t.TargetSpec.Secret.Labels {
				ret.Labels[k] = v
			}
		}
		if t.TargetSpec.Secret.Annotations != nil {
			ret.Annotations = make(map[string]string, len(*t.TargetSpec.Secret.Annotations))
			for k, v := range *t.TargetSpec.Secret.Annotations {
				ret.Annotations[k] = v
			}
		}
		// Overriding of linked SAs not implemented yet...
		// if t.TargetSpec.Secret.LinkedTo != nil {
		// 	ret.LinkedTo = make([]api.SecretLink, 0, len(*t.TargetSpec.Secret.LinkedTo))
		// 	copy(ret.LinkedTo, *t.TargetSpec.Secret.LinkedTo)
		// }
	}

	return *ret
}

func (t *NamespaceTarget) GetClient() client.Client {
	return t.Client
}

func (t *NamespaceTarget) GetTargetObjectKey() client.ObjectKey {
	return t.TargetKey
}

func (t *NamespaceTarget) GetTargetNamespace() string {
	// target spec can be nil if the caller specifically wants to only process existing stuff
	// (e.g. finalizer that just deletes stuff) or if the status and spec are out of sync
	// (e.g. when we reconcile after a user removed a target from the spec of the remote secret).
	// target status is going to be nil if the spec and status are out of sync (e.g. user
	// added stuff to spec).
	if t.TargetSpec != nil {
		return t.TargetSpec.Namespace
	} else if t.TargetStatus != nil {
		return t.TargetStatus.Namespace
	} else {
		// should never happen, but we need to return something
		return ""
	}
}

func (t *NamespaceTarget) GetActualSecretName() string {
	if t.TargetStatus == nil {
		return ""
	} else {
		return t.TargetStatus.Secret.Name
	}
}

func (t *NamespaceTarget) GetActualServiceAccountNames() []string {
	if t.TargetStatus == nil {
		return []string{}
	} else {
		return t.TargetStatus.ServiceAccountNames
	}
}

func (t *NamespaceTarget) GetType() string {
	return "Namespace"
}
