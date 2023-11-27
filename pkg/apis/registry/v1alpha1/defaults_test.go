// Copyright 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
)

var _ = Describe("Defaults", func() {

	var (
		defaultSize = resource.MustParse("10Gi")
	)

	Describe("RegistryCache defaulting", func() {
		It("should default correctly", func() {
			obj := &v1alpha1.RegistryConfig{
				Caches: []v1alpha1.RegistryCache{
					{},
				},
			}

			v1alpha1.SetObjectDefaults_RegistryConfig(obj)

			expected := &v1alpha1.RegistryConfig{
				Caches: []v1alpha1.RegistryCache{
					{
						Size: &defaultSize,
						GarbageCollection: &v1alpha1.GarbageCollection{
							Enabled: true,
						},
					},
				},
			}
			Expect(obj).To(Equal(expected))
		})

		It("should not overwrite already set values", func() {
			customSize := resource.MustParse("20Gi")
			obj := &v1alpha1.RegistryConfig{
				Caches: []v1alpha1.RegistryCache{
					{
						Size: &customSize,
						GarbageCollection: &v1alpha1.GarbageCollection{
							Enabled: false,
						},
					},
				},
			}
			expected := obj.DeepCopy()

			v1alpha1.SetObjectDefaults_RegistryConfig(obj)

			Expect(obj).To(Equal(expected))
		})
	})
})
