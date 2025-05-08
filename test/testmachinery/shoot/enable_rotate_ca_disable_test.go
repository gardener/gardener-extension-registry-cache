// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/test/framework"
	"github.com/gardener/gardener/test/utils/access"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

const (
	rotateCATestTimeout        = 60 * time.Minute
	rotateCATestCleanupTimeout = 20 * time.Minute
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	var (
		f = framework.NewShootFramework(nil)

		isVerticalPodAutoscalerDisabled bool
	)

	f.Serial().CIt("should enable extension, rotate CA, disable extension", func(parentCtx context.Context) {
		Expect(f.ShootClient).NotTo(BeNil(), "Shoot client should not be nil. If it is the Shoot might be hibernated")

		By("Enable the registry-cache extension")
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			size := GetValidVolumeSize(shoot.Spec.Provider.Type, "2Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{Upstream: "ghcr.io", Volume: &v1alpha3.Volume{Size: &size}},
			})

			if v1beta1helper.ShootWantsVerticalPodAutoscaler(f.Shoot) {
				shoot.Spec.Kubernetes.VerticalPodAutoscaler.Enabled = false
				isVerticalPodAutoscalerDisabled = true
			}

			return nil
		})).To(Succeed())

		By("Verify registry-cache works")
		// Images ghcr.io/jitesoft/alpine:3.18.8, ghcr.io/jitesoft/alpine:3.19.4 and ghcr.io/jitesoft/alpine:3.20.3 are already used by the
		// "should enable and disable the registry-cache extension" and "should enable extension, hibernate Shoot, reconcile Shoot, wake up Shoot, disable extension"
		// tests and may be present in the Node.
		// So the current test will use ghcr.io/jitesoft/alpine:3.15.11, ghcr.io/jitesoft/alpine:3.16.9 and ghcr.io/jitesoft/alpine:3.17.9.
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine31511Image, common.AlpinePodMutateFn)

		By("Start CA rotation")
		ctx, cancel = context.WithTimeout(parentCtx, 25*time.Minute)
		defer cancel()
		Expect(f.AnnotateShoot(ctx, f.Shoot, map[string]string{v1beta1constants.GardenerOperation: v1beta1constants.OperationRotateCAStart})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())
		Expect(v1beta1helper.GetShootCARotationPhase(f.Shoot.Status.Credentials)).To(Equal(gardencorev1beta1.RotationPrepared))

		By("Verify registry-cache works when CA rotation phase is Prepared")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine3169Image, common.AlpinePodMutateFn)

		By("Complete CA rotation")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.AnnotateShoot(ctx, f.Shoot, map[string]string{v1beta1constants.GardenerOperation: v1beta1constants.OperationRotateCAComplete})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())
		Expect(v1beta1helper.GetShootCARotationPhase(f.Shoot.Status.Credentials)).To(Equal(gardencorev1beta1.RotationCompleted))

		// After CA rotation, the shoot client must be recreated so that it uses the new CA.
		// Overwrite the test framework's Shoot client as well because on test failure the framework
		// needs to fetch resources and events from the Shoot cluster.
		shootClient, err := access.CreateShootClientFromAdminKubeconfig(ctx, f.GardenClient, f.Shoot)
		Expect(err).NotTo(HaveOccurred())
		f.ShootClient = shootClient

		By("Verify registry-cache works when CA rotation phase is Completed")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine3179Image, common.AlpinePodMutateFn)
	}, rotateCATestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
		if v1beta1helper.GetShootCARotationPhase(f.Shoot.Status.Credentials) == gardencorev1beta1.RotationPrepared {
			Expect(f.AnnotateShoot(ctx, f.Shoot, map[string]string{v1beta1constants.GardenerOperation: v1beta1constants.OperationRotateCAComplete})).To(Succeed())
			Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())
		}
		if common.HasRegistryCacheExtension(f.Shoot) {
			By("Disable the registry-cache extension")
			Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
				common.RemoveExtension(shoot, "registry-cache")

				if isVerticalPodAutoscalerDisabled {
					shoot.Spec.Kubernetes.VerticalPodAutoscaler.Enabled = true
				}

				return nil
			})).To(Succeed())
		}
	}, rotateCATestCleanupTimeout))
})
