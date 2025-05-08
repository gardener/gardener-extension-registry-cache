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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

const (
	hibernationTestTimeout        = 60 * time.Minute
	hibernationTestCleanupTimeout = 25 * time.Minute
)

var _ = Describe("Shoot registry cache testing", func() {
	var (
		f = framework.NewShootFramework(nil)

		isVerticalPodAutoscalerDisabled bool
		isShootHibernated               bool
	)

	f.Serial().CIt("should enable extension, hibernate Shoot, reconcile Shoot, wake up Shoot, disable extension", func(parentCtx context.Context) {
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
		// We are using ghcr.io/jitesoft/alpine:3.19.4 as ghcr.io/jitesoft/alpine:3.18.8 is already used by the "should enable and disable the registry-cache extension" test.
		// Hence, ghcr.io/jitesoft/alpine:3.18.8 will be present in the Node.
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine3194Image, common.AlpinePodMutateFn)

		By("Hibernate Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		isShootHibernated = true
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

		By("Verify registry-cache works after wake up")
		// We are using ghcr.io/jitesoft/alpine:3.20.3 as ghcr.io/jitesoft/alpine:3.19.4 is already used above and already present in the Node and in the registry cache.
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine3203Image, common.AlpinePodMutateFn)
	}, hibernationTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
		if isShootHibernated && v1beta1helper.HibernationIsEnabled(f.Shoot) {
			By("Wake up Shoot")
			Expect(f.WakeUpShoot(ctx)).To(Succeed())
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
	}, hibernationTestCleanupTimeout))
})
