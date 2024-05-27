// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
	purpose := gardencorev1beta1.ShootPurposeTesting

	return &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: generateName,
			Annotations: map[string]string{
				v1beta1constants.AnnotationShootCloudConfigExecutionMaxDelaySeconds: "0",
			},
		},
		Spec: gardencorev1beta1.ShootSpec{
			CloudProfileName:  "local",
			SecretBindingName: ptr.To("local"),
			Region:            "local",
			Purpose:           &purpose,
			Kubernetes: gardencorev1beta1.Kubernetes{
				Version: "1.30.0",
				Kubelet: &gardencorev1beta1.KubeletConfig{
					SerializeImagePulls: ptr.To(false),
					RegistryPullQPS:     ptr.To(int32(10)),
					RegistryBurst:       ptr.To(int32(20)),
				},
				KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{},
			},
			Networking: &gardencorev1beta1.Networking{
				Type: ptr.To("calico"),
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
