//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&MirrorConfig{}, func(obj interface{}) { SetObjectDefaults_MirrorConfig(obj.(*MirrorConfig)) })
	return nil
}

func SetObjectDefaults_MirrorConfig(in *MirrorConfig) {
	for i := range in.Mirrors {
		a := &in.Mirrors[i]
		for j := range a.Hosts {
			b := &a.Hosts[j]
			SetDefaults_MirrorHost(b)
		}
	}
}