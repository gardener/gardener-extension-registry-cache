// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registrycaches

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	kubeapiserverconstants "github.com/gardener/gardener/pkg/component/kubernetes/apiserver/constants"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	scrapeConfig = `- job_name: registry-cache-metrics
  scheme: https
  tls_config:
    ca_file: /etc/prometheus/seed/ca.crt
  authorization:
    type: Bearer
    credentials_file: /var/run/secrets/gardener.cloud/shoot/token/token
  honor_labels: false
  kubernetes_sd_configs:
  - role: pod
    api_server: https://` + v1beta1constants.DeploymentNameKubeAPIServer + `:` + strconv.Itoa(kubeapiserverconstants.Port) + `
    namespaces:
      names: [ kube-system ]
    tls_config:
      ca_file: /etc/prometheus/seed/ca.crt
    authorization:
      type: Bearer
      credentials_file: /var/run/secrets/gardener.cloud/shoot/token/token
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_upstream_host, __meta_kubernetes_pod_container_port_name]
    action: keep
    regex: (.+);debug
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - target_label: __address__
    action: replace
    replacement: ` + v1beta1constants.DeploymentNameKubeAPIServer + `:` + strconv.Itoa(kubeapiserverconstants.Port) + `
  - source_labels: [__meta_kubernetes_pod_name, __meta_kubernetes_pod_container_port_number]
    action: replace
    target_label: __metrics_path__
    regex: (.+);(.+)
    replacement: /api/v1/namespaces/kube-system/pods/${1}:${2}/proxy/metrics
  metric_relabel_configs:
  - source_labels: [ __name__ ]
    regex: registry_proxy_.+
    action: keep
`
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

func (r *registryCaches) scrapeConfig() string {
	return scrapeConfig
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
			v1beta1constants.PrometheusConfigMapScrapeConfig:   r.scrapeConfig(),
			v1beta1constants.PlutonoConfigMapOperatorDashboard: r.dashboard(),
		}

		return nil
	})

	return err
}
