// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	projectNamespace = "garden-local"
)

// DefaultShootCreationFramework returns default Shoot creation framework for e2e tests.
func DefaultShootCreationFramework() *framework.ShootCreationFramework {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	return framework.NewShootCreationFramework(&framework.ShootCreationConfig{
		GardenerConfig: &framework.GardenerConfig{
			ProjectNamespace:   projectNamespace,
			GardenerKubeconfig: kubeconfigPath,
			SkipAccessingShoot: false,
			CommonConfig:       &framework.CommonConfig{},
		},
	})
}

// DefaultShoot returns a Shoot object with default values for the e2e tests.
func DefaultShoot(generateName string) *gardencorev1beta1.Shoot {
	return &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: generateName,
			Annotations: map[string]string{
				v1beta1constants.AnnotationShootCloudConfigExecutionMaxDelaySeconds: "0",
			},
		},
		Spec: gardencorev1beta1.ShootSpec{
			CloudProfile: &gardencorev1beta1.CloudProfileReference{
				Name: "local",
			},
			CredentialsBindingName: ptr.To("local"),
			Region:                 "local",
			Purpose:                ptr.To(gardencorev1beta1.ShootPurposeTesting),
			Kubernetes: gardencorev1beta1.Kubernetes{
				Version: "1.34.3",
				Kubelet: &gardencorev1beta1.KubeletConfig{
					SerializeImagePulls: ptr.To(false),
					RegistryPullQPS:     ptr.To[int32](10),
					RegistryBurst:       ptr.To[int32](20),
				},
				VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
					Enabled: false,
				},
			},
			Networking: &gardencorev1beta1.Networking{
				Type: ptr.To("calico"),
				// Must be within 10.0.0.0/16 (subnet of kind pod CIDR 10.0.0.0/15, but disjoint with seed pod CIDR 10.1.0.0/16).
				Nodes: ptr.To("10.0.0.0/16"),
			},
			Provider: gardencorev1beta1.Provider{
				Type: "local",
				Workers: []gardencorev1beta1.Worker{{
					Name: "local",
					Machine: gardencorev1beta1.Machine{
						Type: "local",
					},
					CRI: &gardencorev1beta1.CRI{
						Name: gardencorev1beta1.CRINameContainerD,
					},
					Minimum: 1,
					Maximum: 1,
				}},
			},
		},
	}
}
