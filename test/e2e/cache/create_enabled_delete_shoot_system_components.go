// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), Ordered, func() {
	f := e2e.DefaultShootCreationFramework()
	shoot := e2e.DefaultShoot("e2e-cache-ssc")
	size := resource.MustParse("2Gi")
	common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
		{Upstream: "europe-docker.pkg.dev", Volume: &v1alpha3.Volume{Size: &size}},
		{Upstream: "quay.io", Volume: &v1alpha3.Volume{Size: &size}},
		{Upstream: "registry.k8s.io", Volume: &v1alpha3.Volume{Size: &size}},
	})
	f.Shoot = shoot

	It("should create Shoot", func(ctx SpecContext) {
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()
	}, SpecTimeout(15*time.Minute))

	It("should verify no I/O timeout logs in containerd for image pulls", func(ctx SpecContext) {
		verifyNoTimeoutLogsInContainerd(ctx, f.Logger, f.ShootFramework.ShootClient)
	}, SpecTimeout(3*time.Minute))

	It("[europe-docker.pkg.dev] should verify registry-cache works", func(ctx SpecContext) {
		common.VerifyRegistryCache(ctx, f.Logger, f.ShootFramework.ShootClient, common.ArtifactRegistryAlpine3243Image, common.AlpinePodMutateFn)
	}, SpecTimeout(12*time.Minute))

	It("[registry.k8s.io] should verify registry-cache works", func(ctx SpecContext) {
		common.VerifyRegistryCache(ctx, f.Logger, f.ShootFramework.ShootClient, common.RegistryK8sNginx1154Image)
	}, SpecTimeout(12*time.Minute))

	It("should delete Shoot", func(ctx SpecContext) {
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	}, SpecTimeout(15*time.Minute))
})

func verifyNoTimeoutLogsInContainerd(ctx context.Context, logger logr.Logger, shootClient kubernetes.Interface) {
	nodeList, err := framework.GetAllNodesInWorkerPool(ctx, shootClient, new("local"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nodeList.Items).ToNot(BeEmpty(), "Expected to find at least one Node in the cluster")

	rootPodExecutor := framework.NewRootPodExecutor(logger, shootClient, &nodeList.Items[0].Name, metav1.NamespaceSystem)
	defer func(ctx context.Context, rootPodExecutor framework.RootPodExecutor) {
		_ = rootPodExecutor.Clean(ctx)
	}(ctx, rootPodExecutor)
	EventuallyWithOffset(1, ctx, func(g Gomega) {
		// Make sure we don't have a Node bootstrap issue, i.e. there is no I/O timeout during image pull in the containerd logs.
		// https://github.com/gardener/gardener-extension-registry-cache/pull/68 fixes the Node bootstrap issue
		// and this test verifies that the scenario does not regress.
		output, err := rootPodExecutor.Execute(ctx, "sh", "-c", `journalctl -u containerd | grep -E "msg=\"trying next host\" error=\"failed to do request: Head .+ i/o timeout\"" || test $? = 1`)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(output)).To(BeEmpty())
	}).WithPolling(10 * time.Second).Should(Succeed())
}
