// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	parentCtx := context.Background()

	f := e2e.DefaultShootCreationFramework()
	shoot := e2e.DefaultShoot("e2e-cache-ssc")
	size := resource.MustParse("2Gi")
	common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
		{Upstream: "europe-docker.pkg.dev", Volume: &v1alpha3.Volume{Size: &size}},
		{Upstream: "quay.io", Volume: &v1alpha3.Volume{Size: &size}},
		{Upstream: "registry.k8s.io", Volume: &v1alpha3.Volume{Size: &size}},
	})
	f.Shoot = shoot

	It("should create Shoot with registry-cache extension enabled with caches for Shoot system components, delete Shoot", func() {
		By("Create Shoot")

		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Make sure there is no I/O timeout during containerd image pulls")
		ctx, cancel = context.WithTimeout(parentCtx, 3*time.Minute)
		defer cancel()

		nodeList, err := framework.GetAllNodesInWorkerPool(ctx, f.ShootFramework.ShootClient, ptr.To("local"))
		Expect(err).NotTo(HaveOccurred())
		Expect(len(nodeList.Items)).To(BeNumerically(">=", 1), "Expected to find at least one Node in the cluster")

		rootPodExecutor := framework.NewRootPodExecutor(f.Logger, f.ShootFramework.ShootClient, &nodeList.Items[0].Name, metav1.NamespaceSystem)
		defer func(ctx context.Context, rootPodExecutor framework.RootPodExecutor) {
			_ = rootPodExecutor.Clean(ctx)
		}(ctx, rootPodExecutor)
		// Make sure we don't have a Node bootstrap issue, i.e. there is no I/O timeout during image pull in the containerd logs.
		// https://github.com/gardener/gardener-extension-registry-cache/pull/68 fixes the Node bootstrap issue
		// and this tests verifies that the scenario does not regress.
		output, err := rootPodExecutor.Execute(ctx, `journalctl -u containerd | grep -E "msg=\"trying next host\" error=\"failed to do request: Head .+ i/o timeout\"" || test $? = 1`)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(output)).To(BeEmpty())

		By("[europe-docker.pkg.dev] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.ArtifactRegistryNginx1176Image)

		By("[registry.k8s.io] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, common.RegistryK8sNginx1154Image)

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
})
