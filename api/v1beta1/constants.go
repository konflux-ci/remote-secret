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
	UploadSecretLabel         = "appstudio.redhat.com/upload-secret"           //#nosec G101 -- false positive, this is just a label
	LinkedByRemoteSecretLabel = "appstudio.redhat.com/linked-by-remote-secret" //#nosec G101 -- false positive, this is just a label

	RemoteSecretNameAnnotation         = "appstudio.redhat.com/remotesecret-name" //#nosec G101 -- false positive
	TargetNamespaceAnnotation          = "appstudio.redhat.com/remotesecret-target-namespace"
	ManagingRemoteSecretNameAnnotation = "appstudio.redhat.com/managing-remote-secret" //#nosec G101 -- false positive
	LinkedRemoteSecretsAnnotation      = "appstudio.redhat.com/linked-remote-secrets"  //#nosec G101 -- false positive

	// RemoteSecretPartialUpdateAnnotation if present on the upload secret, this marks the upload secret as performing a partial update of the already existing secret data
	// of the remote secret that the upload secret refers to using the RemoteSecretNameAnnotation annotation. The value of this annotation is not important but should be documented
	// as "true". The data of the upload secret is used to update the secret data (i.e. the keys from the upload secret overwrite the keys in the secret data (adding new keys if not
	// present in the secret data)).
	RemoteSecretPartialUpdateAnnotation = "appstudio.redhat.com/remotesecret-partial-update"
	// RemoteSecretDeletedKeysAnnotation should be placed on an upload secret if the user want to remove some keys from the secret data of an already existing remote secret. It
	// contains the comma-separated list of keys that should be removed.
	RemoteSecretDeletedKeysAnnotation = "appstudio.redhat.com/remotesecret-deleted-keys"
)
