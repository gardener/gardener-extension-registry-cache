// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"encoding/json"
	"fmt"
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
	registryv1alpha3 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
)

const (
	// PublicEcrAwsNginx1230Image is the public.ecr.aws/nginx/nginx:1.23.0 image.
	PublicEcrAwsNginx1230Image = "public.ecr.aws/nginx/nginx:1.23.0"
	// PublicEcrAwsNginx1240Image is the public.ecr.aws/nginx/nginx:1.24.0 image.
	PublicEcrAwsNginx1240Image = "public.ecr.aws/nginx/nginx:1.24.0"
	// PublicEcrAwsNginx1250Image is the public.ecr.aws/nginx/nginx:1.25.0 image.
	PublicEcrAwsNginx1250Image = "public.ecr.aws/nginx/nginx:1.25.0"

	// ArtifactRegistryNginx1176Image is the europe-docker.pkg.dev/gardener-project/releases/3rd/nginx:1.17.6 image (copy of docker.io/library/nginx:1.17.6).
	ArtifactRegistryNginx1176Image = "europe-docker.pkg.dev/gardener-project/releases/3rd/nginx:1.17.6"
	// RegistryK8sNginx1154Image is the registry.k8s.io/e2e-test-images/nginx:1.15-4 image.
	RegistryK8sNginx1154Image = "registry.k8s.io/e2e-test-images/nginx:1.15-4"
	// GithubRegistryNginx1240Image is the ghcr.io/linuxserver/nginx:1.24.0 image.
	GithubRegistryNginx1240Image = "ghcr.io/linuxserver/nginx:1.24.0"

	// jqExtractRegistryLocation is a jq command that extracts the source location of the '/var/lib/registry' mount from the container's config.json file.
	jqExtractRegistryLocation = `jq -j '.mounts[] | select(.destination=="/var/lib/registry") | .source' /run/containerd/io.containerd.runtime.v2.task/k8s.io/%s/config.json`
	// jqCountManifests is a jq command that counts the number of image manifests in the manifest index.
	// Ref: https://github.com/opencontainers/image-spec/blob/main/image-index.md#example-image-index.
	jqCountManifests = `jq -j '.manifests | length' %s/docker/registry/v2/blobs/%s/data`
)

// AddOrUpdateRegistryCacheExtension adds or updates registry-cache extension with the given caches to the given Shoot.
func AddOrUpdateRegistryCacheExtension(shoot *gardencorev1beta1.Shoot, caches []registryv1alpha3.RegistryCache) {
	providerConfig := &registryv1alpha3.RegistryConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: registryv1alpha3.SchemeGroupVersion.String(),
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

		EventuallyWithOffset(1, ctx, func() string {
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
			EventuallyWithOffset(1, ctx, func() string {
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
		EventuallyWithOffset(2, ctx, func() string {
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
//  3. It verifies that the image is present in the registry's volume.
//     This is a verification that the image pull happened via the registry cache (and the containerd didn't fall back to the upstream).
func VerifyRegistryCache(parentCtx context.Context, log logr.Logger, shootClient kubernetes.Interface, nginxImage string) {
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
					Image: nginxImage,
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

	upstream, path, tag := splitImage(nginxImage)
	selector := labels.SelectorFromSet(labels.Set(map[string]string{"upstream-host": strings.Replace(upstream, ":", "-", 1)}))
	EventuallyWithOffset(1, ctx, func() error {
		registryPod, err := framework.GetFirstRunningPodWithLabels(ctx, selector, metav1.NamespaceSystem, shootClient)
		if err != nil {
			return fmt.Errorf("failed to get a running registry Pod: %w", err)
		}

		rootPodExecutor := framework.NewRootPodExecutor(log, shootClient, &registryPod.Spec.NodeName, metav1.NamespaceSystem)
		defer func(ctx context.Context, rootPodExecutor framework.RootPodExecutor) {
			_ = rootPodExecutor.Clean(ctx)
		}(ctx, rootPodExecutor)

		containerID := strings.TrimPrefix(registryPod.Status.ContainerStatuses[0].ContainerID, "containerd://")
		registryRootPath, err := rootPodExecutor.Execute(ctx, fmt.Sprintf(jqExtractRegistryLocation, containerID))
		if err != nil {
			return fmt.Errorf("failed to extract the source localtion of the '/var/lib/registry' mount from the container's config.json file: %w", err)
		}

		imageDigest, err := rootPodExecutor.Execute(ctx, fmt.Sprintf("cat %s/docker/registry/v2/repositories/%s/_manifests/tags/%s/current/link", string(registryRootPath), path, tag))
		if err != nil {
			return fmt.Errorf("failed to get the %s image digest: %w", nginxImage, err)
		}
		imageSha256Value := strings.TrimPrefix(string(imageDigest), "sha256:")
		imageIndexPath := fmt.Sprintf("sha256/%s/%s", imageSha256Value[:2], imageSha256Value)

		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf(jqCountManifests, string(registryRootPath), imageIndexPath))
		if err != nil {
			return fmt.Errorf("failed to get the %s image index manifests count: %w", nginxImage, err)
		}

		return nil
	}).WithPolling(10*time.Second).Should(Succeed(), "Expected to successfully find the nginx image in the registry's volume")

	By("Delete nginx Pod")
	timeout := 5 * time.Minute
	ctx, cancel = context.WithTimeout(parentCtx, timeout)
	defer cancel()
	ExpectWithOffset(1, framework.DeleteAndWaitForResource(ctx, shootClient, pod, timeout)).To(Succeed())
}

// splitImage splits the image to <upstream>/<path>:<tag>
func splitImage(image string) (upstream, path, tag string) {
	index := strings.Index(image, "/")
	upstream = image[:index]
	path = image[index+1:]
	index = strings.LastIndex(path, ":")
	tag = path[index+1:]
	path = path[:index]
	return
}
