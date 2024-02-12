//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by deepcopy-gen. DO NOT EDIT.

package mirror

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MirrorConfig) DeepCopyInto(out *MirrorConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.Mirrors != nil {
		in, out := &in.Mirrors, &out.Mirrors
		*out = make([]MirrorConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MirrorConfig.
func (in *MirrorConfig) DeepCopy() *MirrorConfig {
	if in == nil {
		return nil
	}
	out := new(MirrorConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MirrorConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MirrorConfiguration) DeepCopyInto(out *MirrorConfiguration) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]MirrorHost, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MirrorConfiguration.
func (in *MirrorConfiguration) DeepCopy() *MirrorConfiguration {
	if in == nil {
		return nil
	}
	out := new(MirrorConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MirrorHost) DeepCopyInto(out *MirrorHost) {
	*out = *in
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = make([]MirrorHostCapability, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MirrorHost.
func (in *MirrorHost) DeepCopy() *MirrorHost {
	if in == nil {
		return nil
	}
	out := new(MirrorHost)
	in.DeepCopyInto(out)
	return out
}
