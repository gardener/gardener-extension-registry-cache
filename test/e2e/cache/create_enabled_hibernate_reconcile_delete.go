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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	parentCtx := context.Background()

	f := e2e.DefaultShootCreationFramework()
	shoot := e2e.DefaultShoot("e2e-cache-hib")
	size := resource.MustParse("2Gi")
	common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
		{Upstream: "ghcr.io", Volume: &v1alpha3.Volume{Size: &size}},
	})
	f.Shoot = shoot

	It("should create Shoot with registry-cache extension enabled, hibernate Shoot, reconcile Shoot, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.GithubRegistryJitesoftAlpine3189Image, common.AlpinePodMutateFn)

		By("Hibernate Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.HibernateShoot(ctx, f.Shoot)).To(Succeed())

		By("Reconcile Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			metav1.SetMetaDataAnnotation(&shoot.ObjectMeta, "gardener.cloud/operation", "reconcile")

			return nil
		})).To(Succeed())
		Expect(f.WaitForShootToBeReconciled(ctx, f.Shoot)).To(Succeed())

		// We cannot properly test "Wake up Shoot" because after a wake up the registry-cache Pod fails to be scheduled because its PVC is bound to already deleted Node.
		//
		// PersistentVolumeClaims in local setup are provisioned by local-path-provisioner. local-path-provisioner creates a hostPath based persistent volume
		// on the Node. The provisioned PV is tightly bound to the Node. When the Node is deleted, the PV's hostPath directory is also deleted.
		// local-path-provisioner does not support moving PVC from one Node to another one (see https://github.com/rancher/local-path-provisioner/issues/31#issuecomment-690772828).
		// A local-path-provisioner PVC cannot deal with Node deletion/rollout.

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
