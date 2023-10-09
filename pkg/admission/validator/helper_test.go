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

package validator_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener-extension-registry-cache/pkg/admission/validator"
)

var _ = Describe("Helpers", func() {

	DescribeTable("#FindRegistryCacheExtension",
		func(extensions []core.Extension, expectedI int, expectedExt core.Extension) {
			i, ext := validator.FindRegistryCacheExtension(extensions)
			Expect(i).To(Equal(expectedI))
			Expect(ext).To(Equal(expectedExt))
		},

		Entry("extensions is nil",
			nil,
			-1, core.Extension{},
		),
		Entry("extensions is empty",
			[]core.Extension{},
			-1, core.Extension{},
		),
		Entry("no registry-cache extension",
			[]core.Extension{
				{Type: "foo"},
				{Type: "bar"},
				{Type: "baz"},
			},
			-1, core.Extension{},
		),
		Entry("with registry-cache extension",
			[]core.Extension{
				{Type: "foo"},
				{Type: "bar"},
				{Type: "registry-cache", ProviderConfig: &runtime.RawExtension{Raw: []byte(`{"one": "two"}`)}},
				{Type: "baz"},
			},
			2, core.Extension{Type: "registry-cache", ProviderConfig: &runtime.RawExtension{Raw: []byte(`{"one": "two"}`)}},
		),
	)
})
