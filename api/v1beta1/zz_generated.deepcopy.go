//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2023.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LinkableSecretSpec) DeepCopyInto(out *LinkableSecretSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.RequiredKeys != nil {
		in, out := &in.RequiredKeys, &out.RequiredKeys
		*out = make([]SecretKey, len(*in))
		copy(*out, *in)
	}
	if in.LinkedTo != nil {
		in, out := &in.LinkedTo, &out.LinkedTo
		*out = make([]SecretLink, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LinkableSecretSpec.
func (in *LinkableSecretSpec) DeepCopy() *LinkableSecretSpec {
	if in == nil {
		return nil
	}
	out := new(LinkableSecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagedServiceAccountSpec) DeepCopyInto(out *ManagedServiceAccountSpec) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagedServiceAccountSpec.
func (in *ManagedServiceAccountSpec) DeepCopy() *ManagedServiceAccountSpec {
	if in == nil {
		return nil
	}
	out := new(ManagedServiceAccountSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RemoteSecret) DeepCopyInto(out *RemoteSecret) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RemoteSecret.
func (in *RemoteSecret) DeepCopy() *RemoteSecret {
	if in == nil {
		return nil
	}
	out := new(RemoteSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RemoteSecret) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RemoteSecretList) DeepCopyInto(out *RemoteSecretList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RemoteSecret, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RemoteSecretList.
func (in *RemoteSecretList) DeepCopy() *RemoteSecretList {
	if in == nil {
		return nil
	}
	out := new(RemoteSecretList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RemoteSecretList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RemoteSecretSpec) DeepCopyInto(out *RemoteSecretSpec) {
	*out = *in
	in.Secret.DeepCopyInto(&out.Secret)
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]RemoteSecretTarget, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RemoteSecretSpec.
func (in *RemoteSecretSpec) DeepCopy() *RemoteSecretSpec {
	if in == nil {
		return nil
	}
	out := new(RemoteSecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RemoteSecretStatus) DeepCopyInto(out *RemoteSecretStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]TargetStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.SecretStatus.DeepCopyInto(&out.SecretStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RemoteSecretStatus.
func (in *RemoteSecretStatus) DeepCopy() *RemoteSecretStatus {
	if in == nil {
		return nil
	}
	out := new(RemoteSecretStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RemoteSecretTarget) DeepCopyInto(out *RemoteSecretTarget) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RemoteSecretTarget.
func (in *RemoteSecretTarget) DeepCopy() *RemoteSecretTarget {
	if in == nil {
		return nil
	}
	out := new(RemoteSecretTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretKey) DeepCopyInto(out *SecretKey) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretKey.
func (in *SecretKey) DeepCopy() *SecretKey {
	if in == nil {
		return nil
	}
	out := new(SecretKey)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretLink) DeepCopyInto(out *SecretLink) {
	*out = *in
	in.ServiceAccount.DeepCopyInto(&out.ServiceAccount)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretLink.
func (in *SecretLink) DeepCopy() *SecretLink {
	if in == nil {
		return nil
	}
	out := new(SecretLink)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretStatus) DeepCopyInto(out *SecretStatus) {
	*out = *in
	if in.Keys != nil {
		in, out := &in.Keys, &out.Keys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretStatus.
func (in *SecretStatus) DeepCopy() *SecretStatus {
	if in == nil {
		return nil
	}
	out := new(SecretStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceAccountLink) DeepCopyInto(out *ServiceAccountLink) {
	*out = *in
	out.Reference = in.Reference
	in.Managed.DeepCopyInto(&out.Managed)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceAccountLink.
func (in *ServiceAccountLink) DeepCopy() *ServiceAccountLink {
	if in == nil {
		return nil
	}
	out := new(ServiceAccountLink)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetStatus) DeepCopyInto(out *TargetStatus) {
	*out = *in
	if in.ServiceAccountNames != nil {
		in, out := &in.ServiceAccountNames, &out.ServiceAccountNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetStatus.
func (in *TargetStatus) DeepCopy() *TargetStatus {
	if in == nil {
		return nil
	}
	out := new(TargetStatus)
	in.DeepCopyInto(out)
	return out
}
