// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	parentCtx := context.Background()

	f := e2e.DefaultShootCreationFramework()
	shoot := e2e.DefaultShoot("e2e-cache-ssc")
	size := resource.MustParse("2Gi")
	common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
		{Upstream: "europe-docker.pkg.dev", Volume: &v1alpha3.Volume{Size: &size}},
		{Upstream: "quay.io", Volume: &v1alpha3.Volume{Size: &size}},
		{Upstream: "registry.k8s.io", Volume: &v1alpha3.Volume{Size: &size}},
	})
	f.Shoot = shoot

	It("should create Shoot with registry-cache extension enabled with caches for Shoot system components, delete Shoot", func() {
		By("Create Shoot")
		// Use 10min as timeout to verify that we don't have a Node bootstrap issue.
		// https://github.com/gardener/gardener-extension-registry-cache/pull/68 fixes the Node bootstrap issue
		// and this tests verifies that the scenario does not regress.
		//
		// Mitigate https://github.com/gardener/gardener-extension-registry-cache/issues/290 by increasing context timeout to 15min to prevent flakes.
		// TODO(dimitar-kostadinov): Fix https://github.com/gardener/gardener-extension-registry-cache/issues/290 in a permanent way.
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("[europe-docker.pkg.dev] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.ArtifactRegistryNginx1176Image)

		By("[registry.k8s.io] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.RegistryK8sNginx1154Image)

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
