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
	shoot := e2e.DefaultShoot("e2e-cache-fd")
	size := resource.MustParse("2Gi")
	common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
		{Upstream: "ghcr.io", Volume: &v1alpha3.Volume{Size: &size}},
	})
	f.Shoot = shoot

	It("should create Shoot with registry-cache extension enabled, force delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.GithubRegistryJitesoftAlpine3189Image, common.AlpinePodMutateFn)

		By("Force Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.ForceDeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
