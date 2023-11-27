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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
)

var _ = Describe("Conversions", func() {

	var (
		size = resource.MustParse("20Gi")

		scheme *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(v1alpha1.AddToScheme(scheme)).To(Succeed())
	})

	Describe("#Convert_v1alpha1_RegistryCache_To_registry_RegistryCache", func() {
		It("should convert successfully", func() {

			in := &v1alpha1.RegistryConfig{
				Caches: []v1alpha1.RegistryCache{
					{
						Upstream: "docker.io",
						Size:     &size,
						GarbageCollection: &v1alpha1.GarbageCollection{
							Enabled: true,
						},
						SecretReferenceName: pointer.String("docker-credentials"),
					},
				},
			}
			out := &registry.RegistryConfig{}

			Expect(scheme.Convert(in, out, nil)).To(Succeed())

			expected := &registry.RegistryConfig{
				Caches: []registry.RegistryCache{
					{
						Upstream: "docker.io",
						Volume: &registry.Volume{
							Size:             &size,
							StorageClassName: pointer.String("default"),
						},
						GarbageCollection: &registry.GarbageCollection{
							Enabled: true,
						},
						SecretReferenceName: pointer.String("docker-credentials"),
					},
				},
			}
			Expect(out).To(Equal(expected))
		})
	})

	Describe("#Convert_registry_RegistryCache_To_v1alpha1_RegistryCache", func() {
		It("should convert successfully", func() {
			in := &registry.RegistryConfig{
				Caches: []registry.RegistryCache{
					{
						Upstream: "docker.io",
						GarbageCollection: &registry.GarbageCollection{
							Enabled: true,
						},
						SecretReferenceName: pointer.String("docker-credentials"),
					},
					{
						Upstream: "quay.io",
						Volume: &registry.Volume{
							Size:             &size,
							StorageClassName: pointer.String("premium"),
						},
					},
				},
			}
			out := &v1alpha1.RegistryConfig{}

			Expect(scheme.Convert(in, out, nil)).To(Succeed())

			expected := &v1alpha1.RegistryConfig{
				Caches: []v1alpha1.RegistryCache{
					{
						Upstream: "docker.io",
						GarbageCollection: &v1alpha1.GarbageCollection{
							Enabled: true,
						},
						SecretReferenceName: pointer.String("docker-credentials"),
					},
					{
						Upstream: "quay.io",
						Size:     &size,
					},
				},
			}
			Expect(out).To(Equal(expected))
		})
	})
})
