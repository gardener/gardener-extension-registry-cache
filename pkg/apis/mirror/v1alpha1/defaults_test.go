// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
)

var _ = Describe("Defaults", func() {

	Describe("MirrorConfig defaulting", func() {
		It("should default correctly", func() {
			obj := &v1alpha1.MirrorConfig{
				Mirrors: []v1alpha1.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []v1alpha1.MirrorHost{
							{
								Host: "https://mirror.gcr.io",
							},
						},
					},
				},
			}

			v1alpha1.SetObjectDefaults_MirrorConfig(obj)

			expected := &v1alpha1.MirrorConfig{
				Mirrors: []v1alpha1.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []v1alpha1.MirrorHost{
							{
								Host:         "https://mirror.gcr.io",
								Capabilities: []v1alpha1.MirrorHostCapability{v1alpha1.MirrorHostCapabilityPull},
							},
						},
					},
				},
			}
			Expect(obj).To(Equal(expected))
		})

		It("should not overwrite already set values", func() {
			obj := &v1alpha1.MirrorConfig{
				Mirrors: []v1alpha1.MirrorConfiguration{
					{
						Upstream: "docker.io",
						Hosts: []v1alpha1.MirrorHost{
							{
								Host:         "https://mirror.gcr.io",
								Capabilities: []v1alpha1.MirrorHostCapability{v1alpha1.MirrorHostCapabilityPull, v1alpha1.MirrorHostCapabilityResolve},
							},
						},
					},
				},
			}
			expected := obj.DeepCopy()

			v1alpha1.SetObjectDefaults_MirrorConfig(obj)

			Expect(obj).To(Equal(expected))
		})
	})
})
