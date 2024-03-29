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

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HeadscaleUser) DeepCopyInto(out *HeadscaleUser) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeadscaleUser.
func (in *HeadscaleUser) DeepCopy() *HeadscaleUser {
	if in == nil {
		return nil
	}
	out := new(HeadscaleUser)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HeadscaleUser) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HeadscaleUserList) DeepCopyInto(out *HeadscaleUserList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HeadscaleUser, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeadscaleUserList.
func (in *HeadscaleUserList) DeepCopy() *HeadscaleUserList {
	if in == nil {
		return nil
	}
	out := new(HeadscaleUserList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HeadscaleUserList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HeadscaleUserSpec) DeepCopyInto(out *HeadscaleUserSpec) {
	*out = *in
	out.PreAuthKey = in.PreAuthKey
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeadscaleUserSpec.
func (in *HeadscaleUserSpec) DeepCopy() *HeadscaleUserSpec {
	if in == nil {
		return nil
	}
	out := new(HeadscaleUserSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HeadscaleUserStatus) DeepCopyInto(out *HeadscaleUserStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeadscaleUserStatus.
func (in *HeadscaleUserStatus) DeepCopy() *HeadscaleUserStatus {
	if in == nil {
		return nil
	}
	out := new(HeadscaleUserStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreAuthKey) DeepCopyInto(out *PreAuthKey) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreAuthKey.
func (in *PreAuthKey) DeepCopy() *PreAuthKey {
	if in == nil {
		return nil
	}
	out := new(PreAuthKey)
	in.DeepCopyInto(out)
	return out
}
