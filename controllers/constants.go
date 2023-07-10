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

package controllers

// Caution: Modifying these constants may have unintended consequences in other projects that rely on remote-secret.
const (
	UploadSecretLabel          = "appstudio.redhat.com/upload-secret"     //#nosec G101 -- false positive, this is not a token
	RemoteSecretNameAnnotation = "appstudio.redhat.com/remotesecret-name" //#nosec G101 -- false positive, this is not a token
	TargetNamespaceAnnotation  = "appstudio.redhat.com/remotesecret-target-namespace"
)
