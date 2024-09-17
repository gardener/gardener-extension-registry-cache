// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"encoding/json"
	"errors"
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
	"sigs.k8s.io/controller-runtime/pkg/client"

	mirrorv1alpha1 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/mirror/v1alpha1"
	registryv1alpha3 "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
)

const (
	// For the e2e tests don't use images from the following upstreams:
	// - docker.io: DockerHub has rate limiting for anonymous users.
	// - gcr.io, registry.k8s.io, quay.io, europe-docker.pkg.dev: These are all registries used in the Gardener's local setup. Avoid using them to do not have conflicts with provider-local in some corner cases.
	// - Amazon ECR: The Distribution project does not support image pulls from Amazon ECR. Ref https://github.com/distribution/distribution/issues/4383.

	// GithubRegistryJitesoftAlpine3189Image is the ghcr.io/jitesoft/alpine:3.18.9 image.
	GithubRegistryJitesoftAlpine3189Image = "ghcr.io/jitesoft/alpine:3.18.9"
	// GithubRegistryJitesoftAlpine3194Image is the ghcr.io/jitesoft/alpine:3.19.4 image.
	GithubRegistryJitesoftAlpine3194Image = "ghcr.io/jitesoft/alpine:3.19.4"
	// GithubRegistryJitesoftAlpine3203Image is the ghcr.io/jitesoft/alpine:3.20.3 image.
	GithubRegistryJitesoftAlpine3203Image = "ghcr.io/jitesoft/alpine:3.20.3"
	// GitlabRegistryJitesoftAlpine31710Image is the registry.gitlab.com/jitesoft/dockerfiles/alpine:3.17.10 image.
	GitlabRegistryJitesoftAlpine31710Image = "registry.gitlab.com/jitesoft/dockerfiles/alpine:3.17.10"

	// ArtifactRegistryNginx1176Image is the europe-docker.pkg.dev/gardener-project/releases/3rd/nginx:1.17.6 image (copy of docker.io/library/nginx:1.17.6).
	ArtifactRegistryNginx1176Image = "europe-docker.pkg.dev/gardener-project/releases/3rd/nginx:1.17.6"
	// RegistryK8sNginx1154Image is the registry.k8s.io/e2e-test-images/nginx:1.15-4 image.
	RegistryK8sNginx1154Image = "registry.k8s.io/e2e-test-images/nginx:1.15-4"

	// jqExtractRegistryLocation is a jq command that extracts the source location of the '/var/lib/registry' mount from the container's config.json file.
	jqExtractRegistryLocation = `jq -j '.mounts[] | select(.destination=="/var/lib/registry") | .source' /run/containerd/io.containerd.runtime.v2.task/k8s.io/%s/config.json`
	// jqExtractManifestDigest is a jq command that extracts the manifest digest for the current OS architecture.
	jqExtractManifestDigest = `jq -j '.manifests[] | select(.platform.architecture=="%s") | .digest' %s/docker/registry/v2/blobs/%s/data`
	// jqExtractLayersDigests is a jq command that extracts layers digests from the manifest.
	// Ref: https://github.com/opencontainers/image-spec/blob/main/manifest.md.
	jqExtractLayersDigests = `jq -r '.layers[].digest' %s/docker/registry/v2/blobs/%s/data`
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

		for _, upstream := range upstreams {
			EventuallyWithOffset(2, ctx, func() string {
				command := fmt.Sprintf("[ -f /etc/containerd/certs.d/%s/hosts.toml ] && echo 'file found' || echo 'file not found'", upstream)
				// err is ignored intentionally to reduce flakes from transient network errors in prow.
				response, _ := rootPodExecutor.Execute(ctx, command)

				return string(response)
			}).WithPolling(10*time.Second).Should(Equal("file not found\n"), fmt.Sprintf("Expected hosts.toml file on node %s for upstream %s to be deleted", node.Name, upstream))
		}

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())
	}
}

// MutatePodFn is an optional function to change the Pod specification depending on the image used.
type MutatePodFn func(pod *corev1.Pod) *corev1.Pod

// SleepInfinity is MutatePodFn that keeps the container running indefinitely.
func SleepInfinity(pod *corev1.Pod) *corev1.Pod {
	pod.Spec.Containers[0].Command = []string{"sleep"}
	pod.Spec.Containers[0].Args = []string{"infinity"}
	return pod
}

// VerifyRegistryCache verifies that a registry cache works as expected.
//
// The verification consists of the following steps:
//  1. It deploys a Pod with the given image.
//  2. It waits until the Pod is running.
//  3. It verifies that the image is present in the registry's volume.
//     This is a verification that the image pull happened via the registry cache (and the containerd didn't fall back to the upstream).
func VerifyRegistryCache(parentCtx context.Context, log logr.Logger, shootClient kubernetes.Interface, image string, mutateFns ...MutatePodFn) {
	upstream, path, tag := splitImage(image)
	name := strings.ReplaceAll(path, "/", "-")
	By(fmt.Sprintf("Create %s Pod", name))
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)
	defer cancel()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name + "-",
			Namespace:    corev1.NamespaceDefault,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  name,
					Image: image,
				},
			},
		},
	}
	for _, mutateFn := range mutateFns {
		pod = mutateFn(pod)
	}
	ExpectWithOffset(1, shootClient.Client().Create(ctx, pod)).To(Succeed())

	By(fmt.Sprintf("Wait until %s Pod is running", name))
	ExpectWithOffset(1, framework.WaitUntilPodIsRunning(ctx, log, pod.Name, pod.Namespace, shootClient)).To(Succeed())

	// get the architecture of Node the Pod is running on
	ExpectWithOffset(1, shootClient.Client().Get(ctx, client.ObjectKeyFromObject(pod), pod)).To(Succeed())
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.Spec.NodeName,
		},
	}
	ExpectWithOffset(1, shootClient.Client().Get(ctx, client.ObjectKeyFromObject(node), node)).To(Succeed())
	arch := node.Status.NodeInfo.Architecture
	log.Info("Node architecture", "arch", arch)

	By(fmt.Sprintf("Verify the registry cache pulled the %s image", image))
	ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
	defer cancel()

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
		log.Info("Registry container ID", "containerID", containerID)
		output, err := rootPodExecutor.Execute(ctx, fmt.Sprintf(jqExtractRegistryLocation, containerID))
		if err != nil {
			log.Error(err, "Failed to extract the source location of the '/var/lib/registry' mount from the container's config.json file", "output", string(output))
			return fmt.Errorf("failed to extract the source location of the '/var/lib/registry' mount from the container's config.json file: command failed with err %w", err)
		}
		registryRootPath := string(output)
		log.Info("Registry root path on node", "registryRootPath", registryRootPath)

		output, err = rootPodExecutor.Execute(ctx, fmt.Sprintf(`cat %s/docker/registry/v2/repositories/%s/_manifests/tags/%s/current/link`, registryRootPath, path, tag))
		if err != nil {
			log.Error(err, "Failed to get the image index digest", "image", image, "output", string(output))
			return fmt.Errorf("failed to get the %s image index digest: %w", image, err)
		}
		imageIndexPath := sha256Path(string(output))
		log.Info("Image index path under <repo-root>/docker/registry/v2/blobs/", "imageIndexPath", imageIndexPath)

		output, err = rootPodExecutor.Execute(ctx, fmt.Sprintf(jqExtractManifestDigest, arch, registryRootPath, imageIndexPath))
		if err != nil {
			log.Error(err, "Failed to get the image manifests digest", "image", image, "output", string(output))
			return fmt.Errorf("failed to get the %s image manifests digest: %w", image, err)
		}
		manifestPath := sha256Path(string(output))
		log.Info("Image manifest path under <repo-root>/docker/registry/v2/blobs/", "image", image, "manifestPath", manifestPath)

		output, err = rootPodExecutor.Execute(ctx, fmt.Sprintf(jqExtractLayersDigests, registryRootPath, manifestPath))
		if err != nil {
			log.Error(err, "Failed to get the image layers digests", "image", image, "output", string(output))
			return fmt.Errorf("failed to get the %s image layers digests: %w", image, err)
		}
		layerDigests := strings.Split(strings.TrimSpace(string(output)), "\n")
		log.Info("Image layers", "count", len(layerDigests))

		var errs []error
		for _, layerDigest := range layerDigests {
			_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf(`[ -f %s/docker/registry/v2/blobs/%s/data ]`, registryRootPath, sha256Path(layerDigest)))
			if err != nil {
				log.Error(err, "Failed to find image layer", "image", image, "digest", layerDigest)
				errs = append(errs, fmt.Errorf("failed to find image %s layer with digest %s", image, layerDigest))
			}
			log.Info("Image layer exists", "image", image, "digest", layerDigest)
		}

		return errors.Join(errs...)
	}).WithPolling(10*time.Second).Should(Succeed(), fmt.Sprintf("Expected to successfully find the %s image in the registry's volume", image))

	By(fmt.Sprintf("Delete %s Pod", name))
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

// sha256Path construct the path under <repo-root>/docker/registry/v2/blobs/
// e.g. sha256:d72807a326fbca3a3bf68a9add2f10248a19205557ddd44b5ad629d8d6c0f805 -> sha256/d7/d72807a326fbca3a3bf68a9add2f10248a19205557ddd44b5ad629d8d6c0f805
func sha256Path(digest string) string {
	if len(digest) < 64 {
		return digest
	}
	value := strings.TrimPrefix(digest, "sha256:")
	return fmt.Sprintf("sha256/%s/%s", value[:2], value)
}
