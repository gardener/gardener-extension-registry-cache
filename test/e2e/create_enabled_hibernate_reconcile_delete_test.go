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

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

var _ = Describe("Registry Cache Extension Tests", func() {
	parentCtx := context.Background()

	f := defaultShootCreationFramework()
	shoot := defaultShoot("e2e-hib")
	size := resource.MustParse("2Gi")
	common.AddRegistryCacheExtension(shoot, []v1alpha1.RegistryCache{
		{Upstream: "docker.io", Size: &size},
	})
	f.Shoot = shoot

	It("should create Shoot with registry-cache extension enabled, hibernate Shoot, reconcile Shoot, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, "docker.io", common.DockerNginx1130ImageWithDigest)

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
