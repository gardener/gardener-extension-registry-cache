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

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
)

const (
	// DockerNginx1130ImageWithDigest corresponds to the nginx:1.13.0 image.
	DockerNginx1130ImageWithDigest = "docker.io/library/nginx@sha256:12d30ce421ad530494d588f87b2328ddc3cae666e77ea1ae5ac3a6661e52cde6"
	// DockerNginx1140ImageWithDigest corresponds to the nginx:1.14.0 image.
	DockerNginx1140ImageWithDigest = "docker.io/library/nginx@sha256:8b600a4d029481cc5b459f1380b30ff6cb98e27544fc02370de836e397e34030"
	// DockerNginx1150ImageWithDigest corresponds to the nginx:1.15.0 image.
	DockerNginx1150ImageWithDigest = "docker.io/library/nginx@sha256:62a095e5da5f977b9f830adaf64d604c614024bf239d21068e4ca826d0d629a4"

	// EuGcrNginx1176ImageWithDigest corresponds to the eu.gcr.io/gardener-project/3rd/nginx:1.17.6 image.
	EuGcrNginx1176ImageWithDigest = "eu.gcr.io/gardener-project/3rd/nginx@sha256:3efdd8ec67f2eb4e96c6f49560f20d6888242f1376808b84ed0ceb064dceae11"
	// RegistryK8sNginx1154ImageWithDigest corresponds to the registry.k8s.io/e2e-test-images/nginx:1.15-4 image.
	RegistryK8sNginx1154ImageWithDigest = "registry.k8s.io/e2e-test-images/nginx@sha256:db048754ae68ae337d8fa96494c96d2a1204c3320f5dcf7e8e71085adec85da6"
	// PublicEcrAwsNginx1199ImageWithDigest corresponds to the public.ecr.aws/nginx/nginx:1.19.9 image.
	PublicEcrAwsNginx1199ImageWithDigest = "public.ecr.aws/nginx/nginx@sha256:f248a8862fa9b21badbe043518f52f143946e2dc705300e8534f8ca624a291b2"
)

// AddOrUpdateRegistryCacheExtension adds or updates registry-cache extension with the given caches to the given Shoot.
func AddOrUpdateRegistryCacheExtension(shoot *gardencorev1beta1.Shoot, caches []v1alpha1.RegistryCache) {
	providerConfig := &v1alpha1.RegistryConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "RegistryConfig",
		},
		Caches: caches,
	}
	providerConfigJSON, err := json.Marshal(&providerConfig)
	utilruntime.Must(err)

	extension := gardencorev1beta1.Extension{
		Type: "registry-cache",
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigJSON,
		},
	}

	i := slices.IndexFunc(shoot.Spec.Extensions, func(ext gardencorev1beta1.Extension) bool {
		return ext.Type == "registry-cache"
	})
	if i == -1 {
		shoot.Spec.Extensions = append(shoot.Spec.Extensions, extension)
	} else {
		shoot.Spec.Extensions[i] = extension
	}
}

// HasRegistryCacheExtension returns whether the Shoot has an extension of type registry-cache.
func HasRegistryCacheExtension(shoot *gardencorev1beta1.Shoot) bool {
	return slices.ContainsFunc(shoot.Spec.Extensions, func(ext gardencorev1beta1.Extension) bool {
		return ext.Type == "registry-cache"
	})
}

// RemoveRegistryCacheExtension removes registry caches extensions from given Shoot.
func RemoveRegistryCacheExtension(shoot *gardencorev1beta1.Shoot) {
	shoot.Spec.Extensions = slices.DeleteFunc(shoot.Spec.Extensions, func(extension gardencorev1beta1.Extension) bool {
		return extension.Type == "registry-cache"
	})
}

// WaitUntilRegistryConfigurationsAreApplied waits until the configure-containerd-registries.service systemd unit gets active on the Nodes.
// The unit will be in activating state until it configures all registry caches.
func WaitUntilRegistryConfigurationsAreApplied(ctx context.Context, log logr.Logger, shootClient kubernetes.Interface) {
	nodes := &corev1.NodeList{}
	ExpectWithOffset(1, shootClient.Client().List(ctx, nodes)).To(Succeed())

	for _, node := range nodes.Items {
		rootPodExecutor := framework.NewRootPodExecutor(log, shootClient, &node.Name, "kube-system")

		EventuallyWithOffset(1, ctx, func(g Gomega) string {
			command := "systemctl is-active configure-containerd-registries.service &>/dev/null && echo 'active' || echo 'not active'"
			// err is ignored intentionally to reduce flakes from transient network errors in prow.
			response, _ := rootPodExecutor.Execute(ctx, command)

			return string(response)
		}).WithPolling(10*time.Second).Should(Equal("active\n"), fmt.Sprintf("Expected the configure-containerd-registries.service unit to be active on node %s", node.Name))

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// VerifyRegistryConfigurationsAreRemoved verifies that configure-containerd-registries.service systemd unit gets deleted (if expectSystemdUnitDeletion is true)
// and the hosts.toml files for the given upstreams are removed.
// The hosts.toml file(s) and the systemd unit are deleted by the registry-configuration-cleaner DaemonSet.
//
// Note that for a Shoot cluster provider-local adds hosts.toml files for localhost:5001, gcr.io, eu.gcr.io, ghcr.io, registry.k8s.io and quay.io.
// Hence, when a registry cache is removed for one of the above upstreams, then provider-local's hosts.toml file will still exist.
func VerifyRegistryConfigurationsAreRemoved(ctx context.Context, log logr.Logger, shootClient kubernetes.Interface, expectSystemdUnitDeletion bool, upstreams []string) {
	nodes := &corev1.NodeList{}
	ExpectWithOffset(1, shootClient.Client().List(ctx, nodes)).To(Succeed())

	for _, node := range nodes.Items {
		rootPodExecutor := framework.NewRootPodExecutor(log, shootClient, &node.Name, "kube-system")

		if expectSystemdUnitDeletion {
			EventuallyWithOffset(1, ctx, func(g Gomega) string {
				command := "systemctl status configure-containerd-registries.service &>/dev/null && echo 'unit found' || echo 'unit not found'"
				// err is ignored intentionally to reduce flakes from transient network errors in prow.
				response, _ := rootPodExecutor.Execute(ctx, command)

				return string(response)
			}).WithPolling(10*time.Second).Should(Equal("unit not found\n"), fmt.Sprintf("Expected the configure-containerd-registries.service systemd unit on node %s to be deleted", node.Name))
		}

		for _, upstream := range upstreams {
			EventuallyWithOffset(1, ctx, func(g Gomega) string {
				command := fmt.Sprintf("[ -f /etc/containerd/certs.d/%s/hosts.toml ] && echo 'file found' || echo 'file not found'", upstream)
				// err is ignored intentionally to reduce flakes from transient network errors in prow.
				response, _ := rootPodExecutor.Execute(ctx, command)

				return string(response)
			}).WithPolling(10*time.Second).Should(Equal("file not found\n"), fmt.Sprintf("Expected hosts.toml file on node %s for upstream %s to be deleted", node.Name, upstream))
		}

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// VerifyRegistryCache verifies that a registry cache works as expected.
//
// The verification consists of the following steps:
//  1. It deploys an nginx Pod with the given image.
//  2. It waits until the Pod is running.
//  3. It verifies that the image is present in the registry's scheduler-state.json file.
//     This is a verification that the image pull happened via the registry cache (and the containerd didn't fall back to the upstream).
func VerifyRegistryCache(parentCtx context.Context, log logr.Logger, shootClient kubernetes.Interface, upstream, nginxImageWithDigest string) {
	By("Create nginx Pod")
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)
	defer cancel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "nginx-",
			Namespace:    corev1.NamespaceDefault,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: nginxImageWithDigest,
				},
			},
		},
	}
	ExpectWithOffset(1, shootClient.Client().Create(ctx, pod)).To(Succeed())

	By("Wait until nginx Pod is running")
	ExpectWithOffset(1, framework.WaitUntilPodIsRunning(ctx, log, pod.Name, pod.Namespace, shootClient)).To(Succeed())

	By("Verify the registry cache pulled the nginx image")
	ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
	defer cancel()

	selector := labels.SelectorFromSet(labels.Set(map[string]string{"upstream-host": upstream}))
	EventuallyWithOffset(1, ctx, func(g Gomega) (err error) {
		reader, err := framework.PodExecByLabel(ctx, selector, "registry-cache", "cat /var/lib/registry/scheduler-state.json", metav1.NamespaceSystem, shootClient)
		if err != nil {
			return fmt.Errorf("failed to cat registry's scheduler-state.json file: %+w", err)
		}

		schedulerStateFileContent, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to read from reader: %+w", err)
		}

		schedulerStateMap := map[string]interface{}{}
		if err := json.Unmarshal(schedulerStateFileContent, &schedulerStateMap); err != nil {
			return fmt.Errorf("failed to unmarshal registry's scheduler-state.json file: %+w", err)
		}

		expectedImage := strings.TrimPrefix(nginxImageWithDigest, upstream+"/")
		if _, ok := schedulerStateMap[expectedImage]; !ok {
			prettyFileContent, _ := json.MarshalIndent(schedulerStateMap, "", "  ")
			return fmt.Errorf("failed to find key (image) '%s' in map (registry's scheduler-state.json file) %v", expectedImage, string(prettyFileContent))
		}

		return nil
	}).WithPolling(10*time.Second).Should(Succeed(), "Expected to successfully find the nginx image in the registry's scheduler-state.json file")

	By("Delete nginx Pod")
	timeout := 5 * time.Minute
	ctx, cancel = context.WithTimeout(parentCtx, timeout)
	defer cancel()
	ExpectWithOffset(1, framework.DeleteAndWaitForResource(ctx, shootClient, pod, timeout)).To(Succeed())
}
