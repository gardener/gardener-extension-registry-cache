// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package helper_test

import (
	"testing"

	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator/helper"
)

func TestHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validator Helper Suite")
}

var _ = Describe("Helpers", func() {

	DescribeTable("#FindExtension",
		func(extensions []core.Extension, extensionType string, expectedI int, expectedExt core.Extension) {
			i, ext := helper.FindExtension(extensions, extensionType)
			Expect(i).To(Equal(expectedI))
			Expect(ext).To(Equal(expectedExt))
		},

		Entry("extensions is nil",
			nil,
			"registry-cache",
			-1, core.Extension{},
		),
		Entry("extensions is empty",
			[]core.Extension{},
			"registry-cache",
			-1, core.Extension{},
		),
		Entry("no registry-cache extension",
			[]core.Extension{
				{Type: "foo"},
				{Type: "bar"},
				{Type: "baz"},
			},
			"registry-cache",
			-1, core.Extension{},
		),
		Entry("with registry-cache extension",
			[]core.Extension{
				{Type: "foo"},
				{Type: "bar"},
				{Type: "registry-cache", ProviderConfig: &runtime.RawExtension{Raw: []byte(`{"one": "two"}`)}},
				{Type: "baz"},
			},
			"registry-cache",
			2, core.Extension{Type: "registry-cache", ProviderConfig: &runtime.RawExtension{Raw: []byte(`{"one": "two"}`)}},
		),
	)
})
