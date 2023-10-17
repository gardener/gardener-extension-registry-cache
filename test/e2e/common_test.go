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

package e2e_test

import (
	"os"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	projectNamespace = "garden-local"
)

func defaultShootCreationFramework() *framework.ShootCreationFramework {
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

func defaultShoot(generateName string) *gardencorev1beta1.Shoot {
	purpose := gardencorev1beta1.ShootPurposeTesting

	return &gardencorev1beta1.Shoot{
		ObjectMeta: metav1.ObjectMeta{
			Name: generateName,
			Annotations: map[string]string{
				v1beta1constants.AnnotationShootInfrastructureCleanupWaitPeriodSeconds: "0",
				v1beta1constants.AnnotationShootCloudConfigExecutionMaxDelaySeconds:    "0",
			},
		},
		Spec: gardencorev1beta1.ShootSpec{
			CloudProfileName:  "local",
			SecretBindingName: pointer.String("local"),
			Region:            "local",
			Purpose:           &purpose,
			Kubernetes: gardencorev1beta1.Kubernetes{
				Version: "1.27.1",
				Kubelet: &gardencorev1beta1.KubeletConfig{
					SerializeImagePulls: pointer.Bool(false),
					RegistryPullQPS:     pointer.Int32(10),
					RegistryBurst:       pointer.Int32(20),
				},
				KubeAPIServer: &gardencorev1beta1.KubeAPIServerConfig{},
				VerticalPodAutoscaler: &gardencorev1beta1.VerticalPodAutoscaler{
					Enabled: true,
				},
			},
			Networking: &gardencorev1beta1.Networking{
				Type: pointer.String("calico"),
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
