// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha2"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

const (
	hibernationTestTimeout        = 60 * time.Minute
	hibernationTestCleanupTimeout = 25 * time.Minute
)

var _ = Describe("Shoot registry cache testing", func() {
	f := framework.NewShootFramework(nil)

	f.Serial().Beta().CIt("should enable extension, hibernate Shoot, reconcile Shoot, wake up Shoot, disable extension", func(parentCtx context.Context) {
		By("Enable the registry-cache extension")
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			if shoot.Spec.Provider.Type == "alicloud" {
				// On AliCloud the minimum size for SSD volumes is 20Gi.
				size = resource.MustParse("20Gi")
			}

			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha2.RegistryCache{
				{Upstream: "docker.io", Volume: &v1alpha2.Volume{Size: &size}},
			})

			return nil
		})).To(Succeed())

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryCacheConfigurationsAreApplied(ctx, f.Logger, f.ShootClient)

		By("Verify registry-cache works")
		// We are using nginx:1.24.0 as nginx:1.23.0 is already used by the "should enable and disable the registry-cache extension" test.
		// Hence, nginx:1.23.0 will be present in the Node.
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, "docker.io", common.DockerNginx1240ImageWithDigest)

		By("Hibernate Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.HibernateShoot(ctx)).To(Succeed())

		By("Reconcile Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, "gardener.cloud/operation", "reconcile")

			return nil
		})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())

		By("Wake up Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.WakeUpShoot(ctx)).To(Succeed())

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryCacheConfigurationsAreApplied(ctx, f.Logger, f.ShootClient)

		By("Verify registry-cache works after wake up")
		// We are using nginx:1.25.0 as nginx:1.24.0 is already used above and already present in the Node and in the registry cache.
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, "docker.io", common.DockerNginx1250ImageWithDigest)
	}, hibernationTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
		if v1beta1helper.HibernationIsEnabled(f.Shoot) {
			By("Wake up Shoot")
			Expect(f.WakeUpShoot(ctx)).To(Succeed())
		}

		if common.HasRegistryCacheExtension(f.Shoot) {
			By("Disable the registry-cache extension")
			Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
				common.RemoveExtension(shoot, "registry-cache")

				return nil
			})).To(Succeed())
		}
	}, hibernationTestCleanupTimeout))
})
