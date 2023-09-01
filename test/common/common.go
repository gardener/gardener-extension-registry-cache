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
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
)

const (
	// Nginx1130ImageWithDigest corresponds to the nginx:1.13.0 image.
	Nginx1130ImageWithDigest = "library/nginx@sha256:12d30ce421ad530494d588f87b2328ddc3cae666e77ea1ae5ac3a6661e52cde6"
	// Nginx1140ImageWithDigest corresponds to the nginx:1.14.0 image.
	Nginx1140ImageWithDigest = "library/nginx@sha256:8b600a4d029481cc5b459f1380b30ff6cb98e27544fc02370de836e397e34030"
	// Nginx1150ImageWithDigest corresponds to the nginx:1.15.0 image.
	Nginx1150ImageWithDigest = "library/nginx@sha256:62a095e5da5f977b9f830adaf64d604c614024bf239d21068e4ca826d0d629a4"
)

// AddRegistryCacheExtension adds registry-cache extension with the given upstream and size to to given Shoot.
func AddRegistryCacheExtension(shoot *gardencorev1beta1.Shoot, upstream string, size *resource.Quantity) {
	providerConfig := &v1alpha1.RegistryConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "RegistryConfig",
		},
		Caches: []v1alpha1.RegistryCache{
			{
				Upstream: upstream,
				Size:     size,
			},
		},
	}
	providerConfigJSON, err := json.Marshal(&providerConfig)
	utilruntime.Must(err)

	extension := gardencorev1beta1.Extension{
		Type: "registry-cache",
		ProviderConfig: &runtime.RawExtension{
			Raw: providerConfigJSON,
		},
	}

	shoot.Spec.Extensions = append(shoot.Spec.Extensions, extension)
}

// RemoveRegistryCacheExtension removes registry-caches extensions from given Shoot.
func RemoveRegistryCacheExtension(shoot *gardencorev1beta1.Shoot) {
	shoot.Spec.Extensions = slices.DeleteFunc(shoot.Spec.Extensions, func(extension gardencorev1beta1.Extension) bool {
		return extension.Type == "registry-cache"
	})
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
	var reader io.Reader
	EventuallyWithOffset(1, ctx, func(g Gomega) (err error) {
		reader, err = framework.PodExecByLabel(ctx, selector, "registry-cache", "cat /var/lib/registry/scheduler-state.json", metav1.NamespaceSystem, shootClient)
		return err
	}).WithPolling(10*time.Second).Should(Succeed(), "Expected to successfully cat registry's scheduler-state.json file")

	schedulerStateFileContent, err := io.ReadAll(reader)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	schedulerStateMap := map[string]interface{}{}
	ExpectWithOffset(1, json.Unmarshal(schedulerStateFileContent, &schedulerStateMap)).To(Succeed())
	ExpectWithOffset(1, schedulerStateMap).To(HaveKey(nginxImageWithDigest), fmt.Sprintf("Expected to find image %s in the registry's scheduler-state.json file", nginxImageWithDigest))

	By("Delete nginx Pod")
	timeout := 5 * time.Minute
	ctx, cancel = context.WithTimeout(parentCtx, timeout)
	defer cancel()
	ExpectWithOffset(1, framework.DeleteAndWaitForResource(ctx, shootClient, pod, timeout)).To(Succeed())
}
