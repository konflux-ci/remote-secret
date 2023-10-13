/*
Copyright 2021.

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

package v1beta1

import (
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RemoteSecretSpec defines the desired state of RemoteSecret
type RemoteSecretSpec struct {
	// Secret defines the properties of the secret and the linked service accounts that should be
	// created in the target namespaces.
	Secret LinkableSecretSpec `json:"secret"`
	// Targets is the list of the target namespaces that the secret and service accounts should be deployed to.
	// +optional
	Targets []RemoteSecretTarget `json:"targets,omitempty"`
}

type RemoteSecretTarget struct {
	// Secret contains the overriden definitions of the secret specific to this target.
	// +kubebuilder:validation:Optional
	Secret *SecretOverride `json:"secret,omitempty"`
	// Namespace is the name of the target namespace to which to deploy.
	Namespace string `json:"namespace"`
	// ApiUrl specifies the URL of the API server of a remote Kubernetes cluster that this target points to. If left empty,
	// the local cluster is assumed.
	// +kubebuilder:validation:Optional
	ApiUrl string `json:"apiUrl,omitempty"`
	// ClusterCredentialsSecret is the name of the secret in the same namespace as the RemoteSecret that contains the token
	// to use to authenticate with the remote Kubernetes cluster. This is ignored if `apiUrl` is empty.
	// +kubebuilder:validation:Optional
	ClusterCredentialsSecret string `json:"clusterCredentialsSecret,omitempty"`
}

type SecretOverride struct {
	// Labels is the new set of labels to be put on the secret instead of the labels defined in the spec. I.e. this completely replaces
	// the labels from the secret spec. Note that this is a pointer to a map so that we can distinguish between an undefined, nil, value
	// and an empty map (clearing any labels defined in the spec).
	// +kubebuilder:validation:Optional
	Labels *map[string]string `json:"labels,omitempty"`
	// Annotations is the new set of annotations to be put on the secret instead of the annotations defined in the spec. I.e. this
	// completely replaces the annotations from the secret spec. Note that this is a pointer to a map so that we can distinguish between
	// an undefined, nil, value and an empty map (clearing any annotations defined in the spec).
	// +kubebuilder:validation:Optional
	Annotations *map[string]string `json:"annotations,omitempty"`
	// LinkedTo is the list of service accounts that the secret will be linked to in the target. This completely replaces the list defined
	// in the secret spec. Note that this is a pointer to an array so that we can distinguish between an undefined, nil, value
	// and an empty array(clearing any links defined in the spec).
	// +kubebuilder:validation:Optional
	LinkedTo *[]SecretLink `json:"linkedTo,omitempty"`
	// Name is the name of the secret when deployed to the target. This overrides the name from the secret spec.
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// GenerateName is the GenerateName of the secret when deployed to the target. This overrides the generateName from the secret spec.
	// +kubebuilder:validation:Optional
	GenerateName string `json:"generateName,omitempty"`
}

// RemoteSecretStatus defines the observed state of RemoteSecret
type RemoteSecretStatus struct {
	// Conditions is the list of conditions describing the state of the deployment
	// to the targets.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Targets is the list of the deployment statuses for individual targets in the spec.
	// +optional
	Targets []TargetStatus `json:"targets,omitempty"`
	// SecretStatus describes the shape of the secret which is currently stored in SecretStorage.
	// +optional
	SecretStatus SecretStatus `json:"secret,omitempty"`
}

type SecretStatus struct {
	Keys []string `json:"keys,omitempty"`
}

type TargetStatus struct {
	// Namespace is the namespace of the target where the secret and the service accounts have been deployed to.
	Namespace string `json:"namespace"`
	// ApiUrl is the URL of the remote Kubernetes cluster to which the target points to.
	ApiUrl string `json:"apiUrl,omitempty"`
	// SecretName is the name of the secret that is actually deployed to the target namespace
	SecretName string `json:"secretName"`
	// ServiceAccountNames is the names of the service accounts that have been deployed to the target namespace
	// +optional
	ServiceAccountNames []string `json:"serviceAccountNames,omitempty"`
	// ClusterCredentialsSecret is the name of the secret in the same namespace as the RemoteSecret that contains the token
	// to use to authenticate with the remote Kubernetes cluster. This is ignored if `apiUrl` is empty.
	ClusterCredentialsSecret string `json:"clusterCredentialsSecret,omitempty"`
	// Error the optional error message if the deployment of either the secret or the service accounts failed.
	// +optional
	Error string `json:"error,omitempty"`
}

// TargetKey is not used in the RemoteSecret spec as such but it represents an identifier of a target. As such it can be used
// as key in the maps instead of RemoteSecretTarget (which itself is not comparable).
//
// Note that it is not directly possible to compare a TargetKey generated from a RemoteSecretTarget and TargetStatus, because
// the spec may contain an empty secret name and non-empty generate name. To determine whether a target key may correspond to
// another, use the CorrespondsTo function on the target key.
type TargetKey struct {
	ApiUrl             string
	Namespace          string
	SecretName         string
	SecretGenerateName string
}

func (ts TargetStatus) ToTargetKey() TargetKey {
	return TargetKey{
		ApiUrl:             ts.ApiUrl,
		Namespace:          ts.Namespace,
		SecretName:         ts.SecretName,
		SecretGenerateName: "",
	}
}

// ToTargetKey converts the remote secret target into a target key given the default spec
// specified in the provided remote secret.
func (rst RemoteSecretTarget) ToTargetKey(containingRemoteSecret *RemoteSecret) TargetKey {
	secretName := ""
	secretGenerateName := ""
	if rst.Secret != nil {
		secretName = rst.Secret.Name
		secretGenerateName = rst.Secret.GenerateName
	}
	if secretName == "" {
		secretName = containingRemoteSecret.Spec.Secret.Name
	}
	if secretGenerateName == "" {
		secretGenerateName = containingRemoteSecret.Spec.Secret.GenerateName
	}
	return TargetKey{
		ApiUrl:             rst.ApiUrl,
		Namespace:          rst.Namespace,
		SecretName:         secretName,
		SecretGenerateName: secretGenerateName,
	}
}

// Correspondence is a result of the TargetKey#CorrespondsTo method describing how two targets correspond to each
// other. Valid values are NameCorrespondence, GenerateNameCorrespondence and NoCorrespondence.
type Correspondence int

const (
	NoCorrespondence Correspondence = iota
	GenerateNameCorrespondence
	NameCorrespondence
)

// CorrespondsTo tells whether the target key corresponds to the other target key. Note that this
// operation is not symmetric, i.e. a.CorrespondsTo(b) does not imply b.CorrespondsTo(a).
func (tk TargetKey) CorrespondsTo(other TargetKey) Correspondence {
	ret := tk.ApiUrl == other.ApiUrl && tk.Namespace == other.Namespace
	if !ret {
		return NoCorrespondence
	}

	// now the fun part...

	if tk.SecretName != "" {
		if tk.SecretName == other.SecretName {
			return NameCorrespondence
		} else {
			return NoCorrespondence
		}
	}

	if tk.SecretGenerateName != "" {
		if other.SecretName != "" {
			if strings.HasPrefix(other.SecretName, tk.SecretGenerateName) {
				return GenerateNameCorrespondence
			} else {
				return NoCorrespondence
			}
		}
	}

	if tk.SecretGenerateName == other.SecretGenerateName {
		return GenerateNameCorrespondence
	} else {
		return NoCorrespondence
	}
}

// RemoteSecretReason is the reconciliation status of the RemoteSecret object
type RemoteSecretReason string

// RemoteSecretConditionType lists the types of conditions we track in the remote secret status
type RemoteSecretConditionType string

const (
	RemoteSecretConditionTypeDeployed     RemoteSecretConditionType = "Deployed"
	RemoteSecretConditionTypeDataObtained RemoteSecretConditionType = "DataObtained"

	RemoteSecretReasonAwaitingTokenData RemoteSecretReason = "AwaitingData"
	RemoteSecretReasonDataFound         RemoteSecretReason = "DataFound"
	RemoteSecretReasonInjected          RemoteSecretReason = "Injected"
	RemoteSecretReasonPartiallyInjected RemoteSecretReason = "PartiallyInjected"
	RemoteSecretReasonError             RemoteSecretReason = "Error"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RemoteSecret is the Schema for the RemoteSecret API
type RemoteSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteSecretSpec   `json:"spec,omitempty"`
	Status RemoteSecretStatus `json:"status,omitempty"`
	// Transient optional field for data to upload during create/update RemoteSecret
	// It is processed by Mutating Webhook and must not be persisted,
	// to make sure (in a case if something happened with Webhook) it is constrained
	//+kubebuilder:validation:MaxProperties=0
	UploadData map[string][]byte `json:"data,omitempty"`
	// Similar to how one can specify the data in an ordinary Kubernetes secret using either
	// the "data" or "stringData" fields, so one can do that when supplying the data to
	// the remote secret. See the data field for more details about the behavior of these fields
	// in remote secrets.
	// Both in create and update, the contents of the stringData is merged into the data field first.
	// This is the same behavior as with ordinary Kubernetes secret's stringData.
	//+kubebuilder:validation:MaxProperties=0
	StringUploadData map[string]string `json:"stringData,omitempty"`
	// DataFrom is an optional field that can be used to copy data from another remote secret during the creation
	// of the remote secret. This field can be specified only during creation of a remote secret (only one of data
	// or dataFrom can be specified at the same time) or during an update of a remote secret that does not yet have
	// data associated with it (its DataObtained condition is in the AwaitingData state).
	DataFrom RemoteSecretDataFrom `json:"dataFrom,omitempty"`
}

var (
	secretTypeMismatchError    = errors.New("the type of upload secret and remote secret spec do not match")
	secretDataKeysMissingError = errors.New("the secret data does not contain the required keys")
)

// ValidateUploadSecret checks whether the uploadSecret type matches the RemoteSecret type and whether upload secret
// contains required keys.
// The function is in the api package because it extends the contract of the CRD.
func (rs *RemoteSecret) ValidateUploadSecret(uploadSecret *corev1.Secret) error {
	if err := checkMatchingSecretTypes(rs.Spec.Secret.Type, uploadSecret.Type); err != nil {
		return err
	}
	if err := rs.ValidateSecretData(uploadSecret.Data); err != nil {
		return err
	}
	return nil
}

// ValidateSecretData checks whether the secret data contains all the keys required by the secret type and specified
// in the RemoteSecret spec. If we assumed this function is called only for upload secrets, we could avoid checking the
// keys required by the secret type because Kubernetes API server would reject the upload secret if it did not contain
// the required keys. However, this function is also meant for validating the secret data that is already stored.
func (rs *RemoteSecret) ValidateSecretData(secretData map[string][]byte) error {
	requiredSetsOfKeys := getKeysForSecretType(rs.Spec.Secret.Type)
	for _, key := range rs.Spec.Secret.RequiredKeys {
		requiredSetsOfKeys = append(requiredSetsOfKeys, []string{key.Name})
	}

	// Check if the secret data contains all the required keys. Outer slice is ANDed, inner slice is ORed.
	var notFoundKeys []string
	for _, keys := range requiredSetsOfKeys {
		// If the current set of required keys is empty, such is the case when type in RemoteSecret is Opaque,
		// then this is an empty constraint, and we can skip the iteration.
		if len(keys) == 0 {
			continue
		}
		found := false
		for _, k := range keys {
			if _, ok := secretData[k]; ok {
				found = true
				break
			}
		}
		if !found {
			notFoundKeys = append(notFoundKeys, strings.Join(keys, " neither "))
		}
	}

	if len(notFoundKeys) > 0 {
		return fmt.Errorf("%w: %s", secretDataKeysMissingError, strings.Join(notFoundKeys, ", "))
	}

	return nil
}

//+kubebuilder:object:root=true

// RemoteSecretList contains a list of RemoteSecret
type RemoteSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteSecret{}, &RemoteSecretList{})
}

type LinkableSecretSpec struct {
	// Name is the name of the secret to be created. If it is not defined a random name based on the name of the binding
	// is used.
	// +optional
	Name         string `json:"name,omitempty"`
	GenerateName string `json:"generateName,omitempty"`
	// Labels contains the labels that the created secret should be labeled with.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations is the keys and values that the created secret should be annotated with.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Type is the type of the secret to be created in targets. If left empty, the default type used in the cluster is assumed (typically Opaque).
	// The Type has to match type of the UploadSecret. This constraint ensures that the requirements on keys,
	// put forth by Kubernetes (https://kubernetes.io/docs/concepts/configuration/secret/#secret-types), are met and
	// secret can be properly created in targets.
	Type corev1.SecretType `json:"type,omitempty"`
	// RequiredKeys are the keys which need to be present in the UploadSecret to successfully upload the SecretData.
	// Furthermore, the UploadSecret needs to contain the keys which are inferred from the Type
	// (and UploadSecret's type, since these have to match) and may contain any additional keys.
	RequiredKeys []SecretKey `json:"keys,omitempty"`
	// LinkedTo specifies the objects that the secret is linked to. Currently, only service accounts are supported.
	LinkedTo []SecretLink `json:"linkedTo,omitempty"`
}
type SecretKey struct {
	Name string `json:"name,omitempty"`
}

type SecretLink struct {
	// ServiceAccounts lists the service accounts that the secret is linked to.
	ServiceAccount ServiceAccountLink `json:"serviceAccount,omitempty"`
}

type ServiceAccountLink struct {
	// As specifies how the secret generated by the binding is linked to the service account.
	// This can be either `secret` meaning that the secret is listed as one of the mountable secrets
	// in the `secrets` of the service account, `imagePullSecret` which makes the secret listed as
	// one of the image pull secrets associated with the service account. If not specified, it defaults
	// to `secret`.
	// +optional
	// +kubebuilder:default:=secret
	As ServiceAccountLinkType `json:"as,omitempty"`
	// Reference specifies a pre-existing service account that the secret should be linked to. It is an error
	// if the service account doesn't exist when the operator tries to add a link to a secret with the injected
	// token.
	Reference corev1.LocalObjectReference `json:"reference,omitempty"`
	// Managed specifies the service account that is bound to the lifetime of the binding. This service account
	// must not exist and is created and deleted along with the injected secret.
	Managed ManagedServiceAccountSpec `json:"managed,omitempty"`
}

type ManagedServiceAccountSpec struct {
	// Name is the name of the service account to create/link. Either this or GenerateName
	// must be specified.
	// +optional
	Name string `json:"name"`
	// GenerateName is the generate name to be used when creating the service account. It only
	// really makes sense for the Managed service accounts that are cleaned up with the binding.
	// +optional
	GenerateName string `json:"generateName"`
	// Labels contains the labels that the created service account should be labeled with.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations is the keys and values that the created service account should be annotated with.
	Annotations map[string]string `json:"annotations,omitempty"`
}
type ServiceAccountLinkType string

const (
	ServiceAccountLinkTypeSecret          ServiceAccountLinkType = "secret"
	ServiceAccountLinkTypeImagePullSecret ServiceAccountLinkType = "imagePullSecret"
)

type RemoteSecretDataFrom struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// EffectiveSecretLinkType returns the secret link type applying the default value if LinkedSecretAs is unspecified by
// the user.
func (s *ServiceAccountLink) EffectiveSecretLinkType() ServiceAccountLinkType {
	if s.As == ServiceAccountLinkTypeImagePullSecret {
		return ServiceAccountLinkTypeImagePullSecret
	}
	return ServiceAccountLinkTypeSecret
}

type RemoteSecretErrorReason string

const (
	RemoteSecretErrorReasonTokenRetrieval RemoteSecretErrorReason = "TokenRetrieval"
	RemoteSecretErrorReasonNoError        RemoteSecretErrorReason = ""
)

func checkMatchingSecretTypes(rsSecretType, uploadSecretType corev1.SecretType) error {
	defaultize := func(secretType corev1.SecretType) corev1.SecretType {
		if secretType == "" {
			return corev1.SecretTypeOpaque
		}
		return secretType
	}

	if defaultize(uploadSecretType) != defaultize(rsSecretType) {
		return fmt.Errorf("%w, uploadSecret: %s, remoteSecret: %s", secretTypeMismatchError, uploadSecretType, rsSecretType)
	}
	return nil
}

// getKeysForSecretType returns the list of keys that are required for a given secret type.
// The mapping is inferred from https://pkg.go.dev/k8s.io/api/core/v1.
// Contract: The outer slice contains elements which should be ANDed together.
// The inner slice contains elements which should be ORed together.
func getKeysForSecretType(secretType corev1.SecretType) [][]string {
	switch secretType {
	case corev1.SecretTypeServiceAccountToken:
		return [][]string{{corev1.ServiceAccountTokenKey}}
	case corev1.SecretTypeDockercfg:
		return [][]string{{corev1.DockerConfigKey}}
	case corev1.SecretTypeDockerConfigJson:
		return [][]string{{corev1.DockerConfigJsonKey}}
	case corev1.SecretTypeBasicAuth:
		return [][]string{{corev1.BasicAuthUsernameKey, corev1.BasicAuthPasswordKey}} // requires at leas one of the keys
	case corev1.SecretTypeSSHAuth:
		return [][]string{{corev1.SSHAuthPrivateKey}}
	case corev1.SecretTypeTLS:
		return [][]string{{corev1.TLSCertKey}, {corev1.TLSPrivateKeyKey}} // requires both keys
	default:
		return [][]string{{}} // Opaque, bootstrap.kubernetes.io/token, and others: no expected keys
	}
}
