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

package v1beta1

// Caution: Modifying these constants may have unintended consequences in other projects that rely on remote-secret.
const (
	UploadSecretLabel                   = "appstudio.redhat.com/upload-secret"           //#nosec G101 -- false positive, this is just a label
	LinkedByRemoteSecretLabel           = "appstudio.redhat.com/linked-by-remote-secret" //#nosec G101 -- false positive, this is just a label
	RemoteSecretAuthServiceAccountLabel = "appstudio.redhat.com/remotesecret-auth-sa"

	RemoteSecretNameAnnotation         = "appstudio.redhat.com/remotesecret-name" //#nosec G101 -- false positive
	TargetNamespaceAnnotation          = "appstudio.redhat.com/remotesecret-target-namespace"
	ManagingRemoteSecretNameAnnotation = "appstudio.redhat.com/managing-remote-secret" //#nosec G101 -- false positive
	LinkedRemoteSecretsAnnotation      = "appstudio.redhat.com/linked-remote-secrets"  //#nosec G101 -- false positive

	// ObjectClusterUrlAnnotation is put on the events that are created when we fail to clean up the deployed secrets during the RemoteSecret
	// finalization. It specifies the API URL of the cluster where the secrets were deployed to (because there is no other good place to put
	// this information on the Event object).
	ObjectClusterUrlAnnotation = "appstudio.redhat.com/object-cluster-url"

	// RemoteSecretPartialUpdateAnnotation if present on the upload secret, this marks the upload secret as performing a partial update of the already existing secret data
	// of the remote secret that the upload secret refers to using the RemoteSecretNameAnnotation annotation. The value of this annotation is not important but should be documented
	// as "true". The data of the upload secret is used to update the secret data (i.e. the keys from the upload secret overwrite the keys in the secret data (adding new keys if not
	// present in the secret data)).
	RemoteSecretPartialUpdateAnnotation = "appstudio.redhat.com/remotesecret-partial-update"
	// RemoteSecretDeletedKeysAnnotation should be placed on an upload secret if the user want to remove some keys from the secret data of an already existing remote secret. It
	// contains the comma-separated list of keys that should be removed.
	RemoteSecretDeletedKeysAnnotation = "appstudio.redhat.com/remotesecret-deleted-keys"

	// EnvironmentNameLabelOrAnnotation is used to specify the name of the environment that the remote secret is associated with.
	// If used as an annotation, it is allows multiple values. If used as a label, it is a single value. Usage is mutually exclusive between label and annotation.
	EnvironmentNameLabelOrAnnotation = "appstudio.redhat.com/environment"
)
