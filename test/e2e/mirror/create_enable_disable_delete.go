// Copyright (c) 2024 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package mirror

import (
	"context"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

var _ = Describe("Registry Mirror Extension Tests", Label("mirror"), func() {
	parentCtx := context.Background()

	f := e2e.DefaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("e2e-mirror-def")

	It("should create Shoot, enable extension, disable extension, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Enable the registry-mirror extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			common.AddOrUpdateRegistryMirrorExtension(shoot, []v1alpha1.MirrorConfiguration{
				{
					Upstream: "docker.io",
					Hosts: []v1alpha1.MirrorHost{
						{Host: "https://mirror.gcr.io"},
					},
				},
				{
					Upstream: "public.ecr.aws",
					Hosts: []v1alpha1.MirrorHost{
						{Host: "https://public-mirror.example.com"},
						{Host: "https://private-mirror.internal", Capabilities: []v1alpha1.MirrorHostCapability{v1alpha1.MirrorHostCapabilityPull, v1alpha1.MirrorHostCapabilityResolve}},
					},
				},
			})

			return nil
		})).To(Succeed())

		By("Verify registry mirror configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 1*time.Minute)
		defer cancel()
		upstreamToHostsTOML := map[string]string{
			"docker.io":      dockerHostsTOML,
			"public.ecr.aws": ecrHostsTOML,
		}
		common.VerifyHostsTOMLFilesCreatedForAllNodes(ctx, f.Logger, f.ShootFramework.ShootClient, upstreamToHostsTOML)

		By("Disable the registry-mirror extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			common.RemoveExtension(shoot, "registry-mirror")

			return nil
		})).To(Succeed())

		By("Verify registry mirror configuration is removed")
		ctx, cancel = context.WithTimeout(parentCtx, 1*time.Minute)
		defer cancel()
		common.VerifyHostsTOMLFilesDeletedForAllNodes(ctx, f.Logger, f.ShootFramework.ShootClient, []string{"docker.io"})

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})

const (
	dockerHostsTOML = `server = "https://registry-1.docker.io"

[host."https://mirror.gcr.io"]
  capabilities = ["pull"]
`

	ecrHostsTOML = `server = "https://public.ecr.aws"

[host."https://public-mirror.example.com"]
  capabilities = ["pull"]

[host."https://private-mirror.internal"]
  capabilities = ["pull", "resolve"]
`
)
