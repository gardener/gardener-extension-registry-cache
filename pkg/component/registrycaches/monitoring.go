// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package registrycaches

import (
	"context"
	_ "embed"
	"fmt"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//go:embed alerting-rules/registry-cache.rules.yaml
	monitoringAlertingRules string
)

func (r *registryCaches) alertingRules() string {
	return fmt.Sprintf("registry-cache.rules.yaml: |\n  %s\n", utils.Indent(monitoringAlertingRules, 2))
}

func (r *registryCaches) deployMonitoringConfigMap(ctx context.Context) error {
	monitoringConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "extension-registry-cache-monitoring",
			Namespace: r.namespace,
		},
	}
	_, err := controllerutils.GetAndCreateOrMergePatch(ctx, r.client, monitoringConfigMap, func() error {
		metav1.SetMetaDataLabel(&monitoringConfigMap.ObjectMeta, v1beta1constants.LabelExtensionConfiguration, v1beta1constants.LabelMonitoring)

		monitoringConfigMap.Data = map[string]string{
			v1beta1constants.PrometheusConfigMapAlertingRules: r.alertingRules(),
		}

		return nil
	})

	return err
}
