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

	mirrorv1alpha1 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
	registryv1alpha1 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
)

const (
	// DockerNginx1230ImageWithDigest corresponds to the nginx:1.23.0 image.
	DockerNginx1230ImageWithDigest = "docker.io/library/nginx@sha256:db345982a2f2a4257c6f699a499feb1d79451a1305e8022f16456ddc3ad6b94c"
	// DockerNginx1240ImageWithDigest corresponds to the nginx:1.24.0 image.
	DockerNginx1240ImageWithDigest = "docker.io/library/nginx@sha256:066476749f229923b9de29cc9a0738ea2d45923b16a2b388449ea549673f97d8"
	// DockerNginx1250ImageWithDigest corresponds to the nginx:1.25.0 image.
	DockerNginx1250ImageWithDigest = "docker.io/library/nginx@sha256:b997b0db9c2bc0a2fb803ced5fb9ff3a757e54903a28ada3e50412cc3ab7822f"

	// ArtifactRegistryNginx1176ImageWithDigest corresponds to the europe-docker.pkg.dev/gardener-project/releases/3rd/nginx:1.17.6 image (copy of docker.io/library/nginx:1.17.6).
	ArtifactRegistryNginx1176ImageWithDigest = "europe-docker.pkg.dev/gardener-project/releases/3rd/nginx@sha256:b2d89d0a210398b4d1120b3e3a7672c16a4ba09c2c4a0395f18b9f7999b768f2"
	// RegistryK8sNginx1154ImageWithDigest corresponds to the registry.k8s.io/e2e-test-images/nginx:1.15-4 image.
	RegistryK8sNginx1154ImageWithDigest = "registry.k8s.io/e2e-test-images/nginx@sha256:db048754ae68ae337d8fa96494c96d2a1204c3320f5dcf7e8e71085adec85da6"
	// PublicEcrAwsNginx1199ImageWithDigest corresponds to the public.ecr.aws/nginx/nginx:1.19.9 image.
	PublicEcrAwsNginx1199ImageWithDigest = "public.ecr.aws/nginx/nginx@sha256:f248a8862fa9b21badbe043518f52f143946e2dc705300e8534f8ca624a291b2"
)

// AddOrUpdateRegistryCacheExtension adds or updates registry-cache extension with the given caches to the given Shoot.
func AddOrUpdateRegistryCacheExtension(shoot *gardencorev1beta1.Shoot, caches []registryv1alpha1.RegistryCache) {
	providerConfig := &registryv1alpha1.RegistryConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: registryv1alpha1.SchemeGroupVersion.String(),
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

// AddOrUpdateRegistryMirrorExtension adds or updates registry-mirror extension with the given mirrors to the given Shoot.
func AddOrUpdateRegistryMirrorExtension(shoot *gardencorev1beta1.Shoot, mirrors []mirrorv1alpha1.MirrorConfiguration) {
	providerConfig := &mirrorv1alpha1.MirrorConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mirrorv1alpha1.SchemeGroupVersion.String(),
			Kind:       "MirrorConfig",
		},
		Mirrors: mirrors,
	}
	providerConfigJSON, err := json.Marshal(&providerConfig)
	utilruntime.Must(err)

	extension := gardencorev1beta1.Extension{
		Type: "registry-mirror",
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigJSON,
		},
	}

	i := slices.IndexFunc(shoot.Spec.Extensions, func(ext gardencorev1beta1.Extension) bool {
		return ext.Type == "registry-mirror"
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

// RemoveExtension removes the extension with the given type from given Shoot.
func RemoveExtension(shoot *gardencorev1beta1.Shoot, extensionType string) {
	shoot.Spec.Extensions = slices.DeleteFunc(shoot.Spec.Extensions, func(extension gardencorev1beta1.Extension) bool {
		return extension.Type == extensionType
	})
}

// WaitUntilRegistryCacheConfigurationsAreApplied waits until the configure-containerd-registries.service systemd unit gets inactive on the Nodes.
// The unit will be in active state until it configures all registry caches.
func WaitUntilRegistryCacheConfigurationsAreApplied(ctx context.Context, log logr.Logger, shootClient kubernetes.Interface) {
	nodes := &corev1.NodeList{}
	ExpectWithOffset(1, shootClient.Client().List(ctx, nodes)).To(Succeed())

	for _, node := range nodes.Items {
		rootPodExecutor := framework.NewRootPodExecutor(log, shootClient, &node.Name, "kube-system")

		EventuallyWithOffset(1, ctx, func(g Gomega) string {
			command := "systemctl -q is-active configure-containerd-registries.service && echo 'active' || echo 'inactive'"
			// err is ignored intentionally to reduce flakes from transient network errors in prow.
			response, _ := rootPodExecutor.Execute(ctx, command)

			return string(response)
		}).WithPolling(10*time.Second).Should(Equal("inactive\n"), fmt.Sprintf("Expected the configure-containerd-registries.service unit to be inactive on node %s", node.Name))

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// VerifyRegistryCacheConfigurationsAreRemoved verifies that configure-containerd-registries.service systemd unit gets deleted (if expectSystemdUnitDeletion is true)
// and the hosts.toml files for the given upstreams are removed.
// The hosts.toml file(s) and the systemd unit are deleted by the registry-configuration-cleaner DaemonSet.
func VerifyRegistryCacheConfigurationsAreRemoved(ctx context.Context, log logr.Logger, shootClient kubernetes.Interface, expectSystemdUnitDeletion bool, upstreams []string) {
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

		VerifyHostsTOMLFilesDeletedForNode(ctx, rootPodExecutor, upstreams, node.Name)

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// VerifyHostsTOMLFilesCreatedForAllNodes verifies that hosts.toml files for the given upstreams are created for all Nodes
// with the given hosts.toml file content.
func VerifyHostsTOMLFilesCreatedForAllNodes(ctx context.Context, log logr.Logger, shootClient kubernetes.Interface, upstreamToHostsTOML map[string]string) {
	nodes := &corev1.NodeList{}
	ExpectWithOffset(1, shootClient.Client().List(ctx, nodes)).To(Succeed())

	for _, node := range nodes.Items {
		rootPodExecutor := framework.NewRootPodExecutor(log, shootClient, &node.Name, "kube-system")

		for upstream, hostsTOML := range upstreamToHostsTOML {
			EventuallyWithOffset(1, ctx, func() string {
				command := fmt.Sprintf("cat /etc/containerd/certs.d/%s/hosts.toml", upstream)
				// err is ignored intentionally to reduce flakes from transient network errors in prow.
				response, _ := rootPodExecutor.Execute(ctx, command)

				return string(response)
			}).WithPolling(10 * time.Second).Should(Equal(hostsTOML))
		}

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// VerifyHostsTOMLFilesDeletedForAllNodes verifies that hosts.toml files for the given upstreams are deleted from all Nodes.
func VerifyHostsTOMLFilesDeletedForAllNodes(ctx context.Context, log logr.Logger, shootClient kubernetes.Interface, upstreams []string) {
	nodes := &corev1.NodeList{}
	ExpectWithOffset(1, shootClient.Client().List(ctx, nodes)).To(Succeed())

	for _, node := range nodes.Items {
		rootPodExecutor := framework.NewRootPodExecutor(log, shootClient, &node.Name, "kube-system")

		VerifyHostsTOMLFilesDeletedForNode(ctx, rootPodExecutor, upstreams, node.Name)

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// VerifyHostsTOMLFilesDeletedForNode verifies that hosts.toml files for the given upstreams are deleted for a Node.
//
// Note that for a Shoot cluster provider-local adds hosts.toml files for localhost:5001, gcr.io, eu.gcr.io, ghcr.io, registry.k8s.io, quay.io and europe-docker.pkg.dev.
// Hence, when a registry cache is removed for one of the above upstreams, then provider-local's hosts.toml file will still exist.
func VerifyHostsTOMLFilesDeletedForNode(ctx context.Context, rootPodExecutor framework.RootPodExecutor, upstreams []string, nodeName string) {
	for _, upstream := range upstreams {
		EventuallyWithOffset(2, ctx, func(g Gomega) string {
			command := fmt.Sprintf("[ -f /etc/containerd/certs.d/%s/hosts.toml ] && echo 'file found' || echo 'file not found'", upstream)
			// err is ignored intentionally to reduce flakes from transient network errors in prow.
			response, _ := rootPodExecutor.Execute(ctx, command)

			return string(response)
		}).WithPolling(10*time.Second).Should(Equal("file not found\n"), fmt.Sprintf("Expected hosts.toml file on node %s for upstream %s to be deleted", nodeName, upstream))
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
