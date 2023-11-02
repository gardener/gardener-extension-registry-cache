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

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
)

var _ = Describe("Registry Cache Extension Tests", func() {
	parentCtx := context.Background()

	f := defaultShootCreationFramework()
	f.Shoot = defaultShoot("e2e-default")

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
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha1.RegistryCache{
				{Upstream: "docker.io", Size: &size},
			})

			return nil
		})).To(Succeed())

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("[docker.io] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, "docker.io", common.DockerNginx1130ImageWithDigest)

		By("Add the public.ecr.aws upstream to the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha1.RegistryCache{
				{Upstream: "docker.io", Size: &size},
				{Upstream: "public.ecr.aws", Size: &size},
			})

			return nil
		})).To(Succeed())

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("[public.ecr.aws] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, "public.ecr.aws", common.PublicEcrAwsNginx1199ImageWithDigest)

		By("Remove the public.ecr.aws upstream from the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			size := resource.MustParse("2Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha1.RegistryCache{
				{Upstream: "docker.io", Size: &size},
			})

			return nil
		})).To(Succeed())

		By("[public.ecr.aws] Verify registry configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		common.VerifyRegistryConfigurationsAreRemoved(ctx, f.Logger, f.ShootFramework.ShootClient, false, []string{"public.ecr.aws"})

		By("Disable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			common.RemoveRegistryCacheExtension(shoot)

			return nil
		})).To(Succeed())

		By("[docker.io] Verify registry configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		common.VerifyRegistryConfigurationsAreRemoved(ctx, f.Logger, f.ShootFramework.ShootClient, true, []string{"docker.io"})

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
