// Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package helper_test

import (
	"testing"

	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1/helper"
)

func TestHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "APIs Registry v1alpha1 Helper Suite")
}

var _ = Describe("Helpers", func() {

	DescribeTable("#GarbageCollectionEnabled",
		func(cache *v1alpha1.RegistryCache, expected bool) {
			Expect(helper.GarbageCollectionEnabled(cache)).To(Equal(expected))
		},
		Entry("garbageCollection is nil", &v1alpha1.RegistryCache{GarbageCollection: nil}, true),
		Entry("garbageCollection.enabled is false", &v1alpha1.RegistryCache{GarbageCollection: &v1alpha1.GarbageCollection{Enabled: false}}, false),
		Entry("garbageCollection.enabled is true", &v1alpha1.RegistryCache{GarbageCollection: &v1alpha1.GarbageCollection{Enabled: true}}, true),
	)

	DescribeTable("#GetSecretReference",
		func(resourceRefs []core.NamedResourceReference, secretReferenceName string, expected *autoscalingv1.CrossVersionObjectReference) {
			Expect(helper.GetSecretReference(resourceRefs, secretReferenceName)).To(Equal(expected))
		},
		Entry("resourceRefs is nil", nil, "foo", nil),
		Entry("resourceRefs is empty", []core.NamedResourceReference{}, "foo", nil),
		Entry("resourceRefs doesn't contains secret ref name", []core.NamedResourceReference{{Name: "bar"}, {Name: "baz"}}, "foo", nil),
		Entry("resourceRefs contains secret ref name with kind ConfigMap",
			[]core.NamedResourceReference{
				{Name: "bar"},
				{Name: "foo", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "ref", Kind: "ConfigMap"}},
			},
			"foo",
			nil),
		Entry("resourceRefs contains secret ref name with kind Secret",
			[]core.NamedResourceReference{
				{Name: "bar"},
				{Name: "foo", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "ref", Kind: "Secret"}},
			},
			"foo",
			&autoscalingv1.CrossVersionObjectReference{Name: "ref", Kind: "Secret"}),
	)
})
