// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/test/framework"
	"github.com/gardener/gardener/test/utils/access"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

const (
	rotateCertTestTimeout        = 60 * time.Minute
	rotateCertTestCleanupTimeout = 20 * time.Minute
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	f := framework.NewShootFramework(nil)

	f.Disruptive().Beta().CIt("should enable extension, rotate Certificate, disable extension", func(parentCtx context.Context) {
		By("Enable the registry-cache extension")
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			if shoot.Spec.Provider.Type == "alicloud" {
				// On AliCloud the minimum size for SSD volumes is 20Gi.
				size = resource.MustParse("20Gi")
			}

			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{Upstream: "ghcr.io", Volume: &v1alpha3.Volume{Size: &size}},
			})

			return nil
		})).To(Succeed())

		By("Verify registry-cache works")
		// Images ghcr.io/jitesoft/alpine:3.18.9, ghcr.io/jitesoft/alpine:3.19.4 and ghcr.io/jitesoft/alpine:3.20.3 are already used by the
		// "should enable and disable the registry-cache extension" and "should enable extension, hibernate Shoot, reconcile Shoot, wake up Shoot, disable extension"
		// tests and may present in the Node.
		// So the current test will use ghcr.io/jitesoft/alpine:3.15.11 ghcr.io/jitesoft/alpine:3.16.9 and ghcr.io/jitesoft/alpine:3.17.9 images.
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine31511Image, common.SleepInfinity)

		By("Rotate certificates phase Preparing")
		ctx, cancel = context.WithTimeout(parentCtx, 25*time.Minute)
		defer cancel()
		Expect(f.AnnotateShoot(ctx, f.Shoot, map[string]string{constants.GardenerOperation: constants.OperationRotateCAStart})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())
		Expect(f.Shoot.Status.Credentials.Rotation.CertificateAuthorities.Phase).To(Equal(gardencorev1beta1.RotationPrepared))

		By("Verify registry-cache works when phase is Prepared")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine3169Image, common.SleepInfinity)

		By("Rotate certificates phase Completing")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.AnnotateShoot(ctx, f.Shoot, map[string]string{constants.GardenerOperation: constants.OperationRotateCAComplete})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())
		Expect(f.Shoot.Status.Credentials.Rotation.CertificateAuthorities.Phase).To(Equal(gardencorev1beta1.RotationCompleted))

		By("Verify registry-cache works after CA rotation")
		// After rotating the CA, the shoot client must be recreated, otherwise any attempt to use it will fail with::
		// "tls: failed to verify certificate: x509: certificate signed by unknown authority"
		shootClient, err := access.CreateShootClientFromAdminKubeconfig(ctx, f.GardenClient, f.Shoot)
		Expect(err).NotTo(HaveOccurred())
		common.VerifyRegistryCache(parentCtx, f.Logger, shootClient, common.GithubRegistryJitesoftAlpine3179Image, common.SleepInfinity)

		By("Disable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			common.RemoveExtension(shoot, "registry-cache")

			return nil
		})).To(Succeed())

		By("Verify registry configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		common.VerifyHostsTOMLFilesDeletedForAllNodes(ctx, f.Logger, shootClient, []string{"ghcr.io"})
	}, rotateCertTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
		if f.Shoot.Status.Credentials != nil && f.Shoot.Status.Credentials.Rotation.CertificateAuthorities.Phase != gardencorev1beta1.RotationCompleted {
			Expect(f.AnnotateShoot(ctx, f.Shoot, map[string]string{constants.GardenerOperation: constants.OperationRotateCAComplete})).To(Succeed())
			Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())
		}
		if common.HasRegistryCacheExtension(f.Shoot) {
			By("Disable the registry-cache extension")
			Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
				common.RemoveExtension(shoot, "registry-cache")

				return nil
			})).To(Succeed())
		}
	}, rotateCertTestCleanupTimeout))
})
