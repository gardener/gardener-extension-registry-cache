// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
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
	f.Shoot = e2e.DefaultShoot("e2e-cache-def")

	It("should create Shoot, enable extension, add upstream, remove upstream, disable extension, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Enable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{Upstream: "docker.io", Volume: &v1alpha3.Volume{Size: &size}},
			})

			return nil
		})).To(Succeed())

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryCacheConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("[docker.io] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.DockerNginx1230Image)

		By("Add the public.ecr.aws upstream to the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{Upstream: "docker.io", Volume: &v1alpha3.Volume{Size: &size}},
				{Upstream: "public.ecr.aws", Volume: &v1alpha3.Volume{Size: &size}},
			})

			return nil
		})).To(Succeed())

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryCacheConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("[public.ecr.aws] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.PublicEcrAwsNginx1199Image)

		By("Remove the public.ecr.aws upstream from the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{Upstream: "docker.io", Volume: &v1alpha3.Volume{Size: &size}},
			})

			return nil
		})).To(Succeed())

		By("[public.ecr.aws] Verify registry configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		common.VerifyRegistryCacheConfigurationsAreRemoved(ctx, f.Logger, f.ShootFramework.ShootClient, false, []string{"public.ecr.aws"})

		By("Disable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			common.RemoveExtension(shoot, "registry-cache")

			return nil
		})).To(Succeed())

		By("[docker.io] Verify registry configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		common.VerifyRegistryCacheConfigurationsAreRemoved(ctx, f.Logger, f.ShootFramework.ShootClient, true, []string{"docker.io"})

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
