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

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

const (
	defaultTestTimeout        = 40 * time.Minute
	defaultTestCleanupTimeout = 10 * time.Minute
)

var _ = Describe("Shoot registry cache testing", func() {
	f := framework.NewShootFramework(nil)

	isVerticalPodAutoscalerDisabled := false

	f.Serial().CIt("should enable and disable the registry-cache extension", func(parentCtx context.Context) {
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
			if v1beta1helper.ShootWantsVerticalPodAutoscaler(f.Shoot) {
				shoot.Spec.Kubernetes.VerticalPodAutoscaler.Enabled = false
				isVerticalPodAutoscalerDisabled = true
			}

			return nil
		})).To(Succeed())

		By("Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootClient, common.GithubRegistryJitesoftAlpine3188Image, common.AlpinePodMutateFn)

		By("Disable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, func(shoot *gardencorev1beta1.Shoot) error {
			common.RemoveExtension(shoot, "registry-cache")
			if isVerticalPodAutoscalerDisabled {
				shoot.Spec.Kubernetes.VerticalPodAutoscaler.Enabled = true
			}

			return nil
		})).To(Succeed())

		By("Verify registry configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		common.VerifyHostsTOMLFilesDeletedForAllNodes(ctx, f.Logger, f.ShootClient, []string{"ghcr.io"})
	}, defaultTestTimeout, framework.WithCAfterTest(func(ctx context.Context) {
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
	}, defaultTestCleanupTimeout))
})
