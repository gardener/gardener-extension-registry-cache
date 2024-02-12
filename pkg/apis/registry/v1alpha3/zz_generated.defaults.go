//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha3

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&RegistryConfig{}, func(obj interface{}) { SetObjectDefaults_RegistryConfig(obj.(*RegistryConfig)) })
	return nil
}

func SetObjectDefaults_RegistryConfig(in *RegistryConfig) {
	for i := range in.Caches {
		a := &in.Caches[i]
		SetDefaults_RegistryCache(a)
		if a.Volume != nil {
			SetDefaults_Volume(a.Volume)
		}
	}
}
