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

package e2e_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

var _ = Describe("Registry Cache Extension Tests", func() {
	parentCtx := context.Background()

	f := defaultShootCreationFramework()
	shoot := defaultShoot("e2e-default-ssc")
	size := resource.MustParse("2Gi")
	common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha1.RegistryCache{
		{Upstream: "eu.gcr.io", Size: &size},
		{Upstream: "quay.io", Size: &size},
		{Upstream: "registry.k8s.io", Size: &size},
	})
	f.Shoot = shoot

	It("should create Shoot with registry-cache extension enabled with caches for Shoot system components, delete Shoot", func() {
		By("Create Shoot")
		// Use 10min as timeout to verify that we don't have a Node bootstrap issue.
		// https://github.com/gardener/gardener-extension-registry-cache/pull/68 fixes the Node bootstrap issue
		// and this tests verifies that the scenario does not regress.
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("[eu.gcr.io] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, "eu.gcr.io", common.EuGcrNginx1176ImageWithDigest)

		By("[registry.k8s.io] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, "registry.k8s.io", common.RegistryK8sNginx1154ImageWithDigest)

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
