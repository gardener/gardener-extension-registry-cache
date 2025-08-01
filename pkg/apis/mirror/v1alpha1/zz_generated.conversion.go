//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	mirror "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*MirrorConfig)(nil), (*mirror.MirrorConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MirrorConfig_To_mirror_MirrorConfig(a.(*MirrorConfig), b.(*mirror.MirrorConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*mirror.MirrorConfig)(nil), (*MirrorConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_mirror_MirrorConfig_To_v1alpha1_MirrorConfig(a.(*mirror.MirrorConfig), b.(*MirrorConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MirrorConfiguration)(nil), (*mirror.MirrorConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MirrorConfiguration_To_mirror_MirrorConfiguration(a.(*MirrorConfiguration), b.(*mirror.MirrorConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*mirror.MirrorConfiguration)(nil), (*MirrorConfiguration)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_mirror_MirrorConfiguration_To_v1alpha1_MirrorConfiguration(a.(*mirror.MirrorConfiguration), b.(*MirrorConfiguration), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MirrorHost)(nil), (*mirror.MirrorHost)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MirrorHost_To_mirror_MirrorHost(a.(*MirrorHost), b.(*mirror.MirrorHost), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*mirror.MirrorHost)(nil), (*MirrorHost)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_mirror_MirrorHost_To_v1alpha1_MirrorHost(a.(*mirror.MirrorHost), b.(*MirrorHost), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_MirrorConfig_To_mirror_MirrorConfig(in *MirrorConfig, out *mirror.MirrorConfig, s conversion.Scope) error {
	out.Mirrors = *(*[]mirror.MirrorConfiguration)(unsafe.Pointer(&in.Mirrors))
	return nil
}

// Convert_v1alpha1_MirrorConfig_To_mirror_MirrorConfig is an autogenerated conversion function.
func Convert_v1alpha1_MirrorConfig_To_mirror_MirrorConfig(in *MirrorConfig, out *mirror.MirrorConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_MirrorConfig_To_mirror_MirrorConfig(in, out, s)
}

func autoConvert_mirror_MirrorConfig_To_v1alpha1_MirrorConfig(in *mirror.MirrorConfig, out *MirrorConfig, s conversion.Scope) error {
	out.Mirrors = *(*[]MirrorConfiguration)(unsafe.Pointer(&in.Mirrors))
	return nil
}

// Convert_mirror_MirrorConfig_To_v1alpha1_MirrorConfig is an autogenerated conversion function.
func Convert_mirror_MirrorConfig_To_v1alpha1_MirrorConfig(in *mirror.MirrorConfig, out *MirrorConfig, s conversion.Scope) error {
	return autoConvert_mirror_MirrorConfig_To_v1alpha1_MirrorConfig(in, out, s)
}

func autoConvert_v1alpha1_MirrorConfiguration_To_mirror_MirrorConfiguration(in *MirrorConfiguration, out *mirror.MirrorConfiguration, s conversion.Scope) error {
	out.Upstream = in.Upstream
	out.Hosts = *(*[]mirror.MirrorHost)(unsafe.Pointer(&in.Hosts))
	return nil
}

// Convert_v1alpha1_MirrorConfiguration_To_mirror_MirrorConfiguration is an autogenerated conversion function.
func Convert_v1alpha1_MirrorConfiguration_To_mirror_MirrorConfiguration(in *MirrorConfiguration, out *mirror.MirrorConfiguration, s conversion.Scope) error {
	return autoConvert_v1alpha1_MirrorConfiguration_To_mirror_MirrorConfiguration(in, out, s)
}

func autoConvert_mirror_MirrorConfiguration_To_v1alpha1_MirrorConfiguration(in *mirror.MirrorConfiguration, out *MirrorConfiguration, s conversion.Scope) error {
	out.Upstream = in.Upstream
	out.Hosts = *(*[]MirrorHost)(unsafe.Pointer(&in.Hosts))
	return nil
}

// Convert_mirror_MirrorConfiguration_To_v1alpha1_MirrorConfiguration is an autogenerated conversion function.
func Convert_mirror_MirrorConfiguration_To_v1alpha1_MirrorConfiguration(in *mirror.MirrorConfiguration, out *MirrorConfiguration, s conversion.Scope) error {
	return autoConvert_mirror_MirrorConfiguration_To_v1alpha1_MirrorConfiguration(in, out, s)
}

func autoConvert_v1alpha1_MirrorHost_To_mirror_MirrorHost(in *MirrorHost, out *mirror.MirrorHost, s conversion.Scope) error {
	out.Host = in.Host
	out.Capabilities = *(*[]mirror.MirrorHostCapability)(unsafe.Pointer(&in.Capabilities))
	return nil
}

// Convert_v1alpha1_MirrorHost_To_mirror_MirrorHost is an autogenerated conversion function.
func Convert_v1alpha1_MirrorHost_To_mirror_MirrorHost(in *MirrorHost, out *mirror.MirrorHost, s conversion.Scope) error {
	return autoConvert_v1alpha1_MirrorHost_To_mirror_MirrorHost(in, out, s)
}

func autoConvert_mirror_MirrorHost_To_v1alpha1_MirrorHost(in *mirror.MirrorHost, out *MirrorHost, s conversion.Scope) error {
	out.Host = in.Host
	out.Capabilities = *(*[]MirrorHostCapability)(unsafe.Pointer(&in.Capabilities))
	return nil
}

// Convert_mirror_MirrorHost_To_v1alpha1_MirrorHost is an autogenerated conversion function.
func Convert_mirror_MirrorHost_To_v1alpha1_MirrorHost(in *mirror.MirrorHost, out *MirrorHost, s conversion.Scope) error {
	return autoConvert_mirror_MirrorHost_To_v1alpha1_MirrorHost(in, out, s)
}
