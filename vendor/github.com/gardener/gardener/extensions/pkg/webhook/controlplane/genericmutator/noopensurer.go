// Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package genericmutator

import (
	"context"

	"github.com/Masterminds/semver/v3"
	"github.com/coreos/go-systemd/v22/unit"
	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"

	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

// NoopEnsurer provides no-op implementation of Ensurer. This can be anonymously composed by actual Ensurers for convenience.
type NoopEnsurer struct{}

var _ Ensurer = &NoopEnsurer{}

// EnsureKubeAPIServerService ensures that the kube-apiserver service conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeAPIServerService(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *corev1.Service) error {
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeAPIServerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *appsv1.Deployment) error {
	return nil
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeControllerManagerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *appsv1.Deployment) error {
	return nil
}

// EnsureKubeSchedulerDeployment ensures that the kube-scheduler deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeSchedulerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *appsv1.Deployment) error {
	return nil
}

// EnsureMachineControllerManagerDeployment ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureMachineControllerManagerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *appsv1.Deployment) error {
	return nil
}

// EnsureMachineControllerManagerVPA ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureMachineControllerManagerVPA(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *vpaautoscalingv1.VerticalPodAutoscaler) error {
	return nil
}

// EnsureClusterAutoscalerDeployment ensures that the cluster-autoscaler deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureClusterAutoscalerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *appsv1.Deployment) error {
	return nil
}

// EnsureETCD ensures that the etcd stateful sets conform to the provider requirements.
func (e *NoopEnsurer) EnsureETCD(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *druidv1alpha1.Etcd) error {
	return nil
}

// EnsureVPNSeedServerDeployment ensures that the vpn-seed-server deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureVPNSeedServerDeployment(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *appsv1.Deployment) error {
	return nil
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *NoopEnsurer) EnsureKubeletServiceUnitOptions(_ context.Context, _ extensionscontextwebhook.GardenContext, _ *semver.Version, new, _ []*unit.UnitOption) ([]*unit.UnitOption, error) {
	return new, nil
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeletConfiguration(_ context.Context, _ extensionscontextwebhook.GardenContext, _ *semver.Version, _, _ *kubeletconfigv1beta1.KubeletConfiguration) error {
	return nil
}

// ShouldProvisionKubeletCloudProviderConfig returns if the cloud provider config file should be added to the kubelet configuration.
func (e *NoopEnsurer) ShouldProvisionKubeletCloudProviderConfig(_ context.Context, _ extensionscontextwebhook.GardenContext, _ *semver.Version) bool {
	return false
}

// EnsureKubeletCloudProviderConfig ensures that the cloud provider config file conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeletCloudProviderConfig(_ context.Context, _ extensionscontextwebhook.GardenContext, _ *semver.Version, _ *string, _ string) error {
	return nil
}

// EnsureKubernetesGeneralConfiguration ensures that the kubernetes general configuration conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubernetesGeneralConfiguration(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *string) error {
	return nil
}

// EnsureAdditionalUnits ensures that additional required system units are added.
func (e *NoopEnsurer) EnsureAdditionalUnits(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *[]extensionsv1alpha1.Unit) error {
	return nil
}

// EnsureAdditionalFiles ensures that additional required system files are added.
func (e *NoopEnsurer) EnsureAdditionalFiles(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *[]extensionsv1alpha1.File) error {
	return nil
}

// EnsureAdditionalProvisionUnits ensures that additional required system units are added.
func (e *NoopEnsurer) EnsureAdditionalProvisionUnits(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *[]extensionsv1alpha1.Unit) error {
	return nil
}

// EnsureAdditionalProvisionFiles ensures that additional required system files are added.
func (e *NoopEnsurer) EnsureAdditionalProvisionFiles(_ context.Context, _ extensionscontextwebhook.GardenContext, _, _ *[]extensionsv1alpha1.File) error {
	return nil
}
