// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
	//go:embed monitoring/dashboard.json
	dashboard string
)

func (r *registryCaches) alertingRules() string {
	return fmt.Sprintf("registry-cache.rules.yaml: |\n  %s\n", utils.Indent(monitoringAlertingRules, 2))
}

func (r *registryCaches) dashboard() string {
	return fmt.Sprintf("registry-cache.dashboard.json: '%s'", dashboard)
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
			v1beta1constants.PrometheusConfigMapAlertingRules:  r.alertingRules(),
			v1beta1constants.PlutonoConfigMapOperatorDashboard: r.dashboard(),
		}

		return nil
	})

	return err
}
