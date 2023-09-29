//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	registry "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	resource "k8s.io/apimachinery/pkg/api/resource"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*GarbageCollection)(nil), (*registry.GarbageCollection)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_GarbageCollection_To_registry_GarbageCollection(a.(*GarbageCollection), b.(*registry.GarbageCollection), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registry.GarbageCollection)(nil), (*GarbageCollection)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registry_GarbageCollection_To_v1alpha1_GarbageCollection(a.(*registry.GarbageCollection), b.(*GarbageCollection), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RegistryCache)(nil), (*registry.RegistryCache)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RegistryCache_To_registry_RegistryCache(a.(*RegistryCache), b.(*registry.RegistryCache), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registry.RegistryCache)(nil), (*RegistryCache)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registry_RegistryCache_To_v1alpha1_RegistryCache(a.(*registry.RegistryCache), b.(*RegistryCache), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RegistryCacheStatus)(nil), (*registry.RegistryCacheStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RegistryCacheStatus_To_registry_RegistryCacheStatus(a.(*RegistryCacheStatus), b.(*registry.RegistryCacheStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registry.RegistryCacheStatus)(nil), (*RegistryCacheStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registry_RegistryCacheStatus_To_v1alpha1_RegistryCacheStatus(a.(*registry.RegistryCacheStatus), b.(*RegistryCacheStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RegistryConfig)(nil), (*registry.RegistryConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RegistryConfig_To_registry_RegistryConfig(a.(*RegistryConfig), b.(*registry.RegistryConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registry.RegistryConfig)(nil), (*RegistryConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registry_RegistryConfig_To_v1alpha1_RegistryConfig(a.(*registry.RegistryConfig), b.(*RegistryConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RegistryStatus)(nil), (*registry.RegistryStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RegistryStatus_To_registry_RegistryStatus(a.(*RegistryStatus), b.(*registry.RegistryStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*registry.RegistryStatus)(nil), (*RegistryStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_registry_RegistryStatus_To_v1alpha1_RegistryStatus(a.(*registry.RegistryStatus), b.(*RegistryStatus), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_GarbageCollection_To_registry_GarbageCollection(in *GarbageCollection, out *registry.GarbageCollection, s conversion.Scope) error {
	out.Enabled = in.Enabled
	return nil
}

// Convert_v1alpha1_GarbageCollection_To_registry_GarbageCollection is an autogenerated conversion function.
func Convert_v1alpha1_GarbageCollection_To_registry_GarbageCollection(in *GarbageCollection, out *registry.GarbageCollection, s conversion.Scope) error {
	return autoConvert_v1alpha1_GarbageCollection_To_registry_GarbageCollection(in, out, s)
}

func autoConvert_registry_GarbageCollection_To_v1alpha1_GarbageCollection(in *registry.GarbageCollection, out *GarbageCollection, s conversion.Scope) error {
	out.Enabled = in.Enabled
	return nil
}

// Convert_registry_GarbageCollection_To_v1alpha1_GarbageCollection is an autogenerated conversion function.
func Convert_registry_GarbageCollection_To_v1alpha1_GarbageCollection(in *registry.GarbageCollection, out *GarbageCollection, s conversion.Scope) error {
	return autoConvert_registry_GarbageCollection_To_v1alpha1_GarbageCollection(in, out, s)
}

func autoConvert_v1alpha1_RegistryCache_To_registry_RegistryCache(in *RegistryCache, out *registry.RegistryCache, s conversion.Scope) error {
	out.Upstream = in.Upstream
	out.Size = (*resource.Quantity)(unsafe.Pointer(in.Size))
	out.GarbageCollection = (*registry.GarbageCollection)(unsafe.Pointer(in.GarbageCollection))
	return nil
}

// Convert_v1alpha1_RegistryCache_To_registry_RegistryCache is an autogenerated conversion function.
func Convert_v1alpha1_RegistryCache_To_registry_RegistryCache(in *RegistryCache, out *registry.RegistryCache, s conversion.Scope) error {
	return autoConvert_v1alpha1_RegistryCache_To_registry_RegistryCache(in, out, s)
}

func autoConvert_registry_RegistryCache_To_v1alpha1_RegistryCache(in *registry.RegistryCache, out *RegistryCache, s conversion.Scope) error {
	out.Upstream = in.Upstream
	out.Size = (*resource.Quantity)(unsafe.Pointer(in.Size))
	out.GarbageCollection = (*GarbageCollection)(unsafe.Pointer(in.GarbageCollection))
	return nil
}

// Convert_registry_RegistryCache_To_v1alpha1_RegistryCache is an autogenerated conversion function.
func Convert_registry_RegistryCache_To_v1alpha1_RegistryCache(in *registry.RegistryCache, out *RegistryCache, s conversion.Scope) error {
	return autoConvert_registry_RegistryCache_To_v1alpha1_RegistryCache(in, out, s)
}

func autoConvert_v1alpha1_RegistryCacheStatus_To_registry_RegistryCacheStatus(in *RegistryCacheStatus, out *registry.RegistryCacheStatus, s conversion.Scope) error {
	out.Upstream = in.Upstream
	out.Endpoint = in.Endpoint
	return nil
}

// Convert_v1alpha1_RegistryCacheStatus_To_registry_RegistryCacheStatus is an autogenerated conversion function.
func Convert_v1alpha1_RegistryCacheStatus_To_registry_RegistryCacheStatus(in *RegistryCacheStatus, out *registry.RegistryCacheStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_RegistryCacheStatus_To_registry_RegistryCacheStatus(in, out, s)
}

func autoConvert_registry_RegistryCacheStatus_To_v1alpha1_RegistryCacheStatus(in *registry.RegistryCacheStatus, out *RegistryCacheStatus, s conversion.Scope) error {
	out.Upstream = in.Upstream
	out.Endpoint = in.Endpoint
	return nil
}

// Convert_registry_RegistryCacheStatus_To_v1alpha1_RegistryCacheStatus is an autogenerated conversion function.
func Convert_registry_RegistryCacheStatus_To_v1alpha1_RegistryCacheStatus(in *registry.RegistryCacheStatus, out *RegistryCacheStatus, s conversion.Scope) error {
	return autoConvert_registry_RegistryCacheStatus_To_v1alpha1_RegistryCacheStatus(in, out, s)
}

func autoConvert_v1alpha1_RegistryConfig_To_registry_RegistryConfig(in *RegistryConfig, out *registry.RegistryConfig, s conversion.Scope) error {
	out.Caches = *(*[]registry.RegistryCache)(unsafe.Pointer(&in.Caches))
	return nil
}

// Convert_v1alpha1_RegistryConfig_To_registry_RegistryConfig is an autogenerated conversion function.
func Convert_v1alpha1_RegistryConfig_To_registry_RegistryConfig(in *RegistryConfig, out *registry.RegistryConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_RegistryConfig_To_registry_RegistryConfig(in, out, s)
}

func autoConvert_registry_RegistryConfig_To_v1alpha1_RegistryConfig(in *registry.RegistryConfig, out *RegistryConfig, s conversion.Scope) error {
	out.Caches = *(*[]RegistryCache)(unsafe.Pointer(&in.Caches))
	return nil
}

// Convert_registry_RegistryConfig_To_v1alpha1_RegistryConfig is an autogenerated conversion function.
func Convert_registry_RegistryConfig_To_v1alpha1_RegistryConfig(in *registry.RegistryConfig, out *RegistryConfig, s conversion.Scope) error {
	return autoConvert_registry_RegistryConfig_To_v1alpha1_RegistryConfig(in, out, s)
}

func autoConvert_v1alpha1_RegistryStatus_To_registry_RegistryStatus(in *RegistryStatus, out *registry.RegistryStatus, s conversion.Scope) error {
	out.Caches = *(*[]registry.RegistryCacheStatus)(unsafe.Pointer(&in.Caches))
	return nil
}

// Convert_v1alpha1_RegistryStatus_To_registry_RegistryStatus is an autogenerated conversion function.
func Convert_v1alpha1_RegistryStatus_To_registry_RegistryStatus(in *RegistryStatus, out *registry.RegistryStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_RegistryStatus_To_registry_RegistryStatus(in, out, s)
}

func autoConvert_registry_RegistryStatus_To_v1alpha1_RegistryStatus(in *registry.RegistryStatus, out *RegistryStatus, s conversion.Scope) error {
	out.Caches = *(*[]RegistryCacheStatus)(unsafe.Pointer(&in.Caches))
	return nil
}

// Convert_registry_RegistryStatus_To_v1alpha1_RegistryStatus is an autogenerated conversion function.
func Convert_registry_RegistryStatus_To_v1alpha1_RegistryStatus(in *registry.RegistryStatus, out *RegistryStatus, s conversion.Scope) error {
	return autoConvert_registry_RegistryStatus_To_v1alpha1_RegistryStatus(in, out, s)
}
