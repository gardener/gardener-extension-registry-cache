package controller

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/gardener/gardener/pkg/utils/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type criEnsurer struct {
	Namespace string
	Labels    map[string]string

	CRIEnsurerImage *imagevector.Image

	ReferencedServices *corev1.ServiceList
}

const (
	criEnsurerName  = "cri-config-ensurer"
	reconcileScript = `
#!/usr/bin/env bash

set -euo pipefail

CONTAINERD_IMPORTS_DIR="/etc/containerd/conf.d"
CONFIG_INPUT_FILE="$1"
TARGET_FILE="/host$CONTAINERD_IMPORTS_DIR/$(basename "$CONFIG_INPUT_FILE")"

if ! grep -F '/etc/containerd/conf.d/*.toml' /host/etc/containerd/config.toml >/dev/null ; then
	# https://github.com/gardener/gardener/blob/v1.51.0/docs/usage/custom-containerd-config.md
	echo "ERROR: Only works on workers created with Gardener >v1.51, exiting."
	exit 1
fi

if [ ! -e "$CONFIG_INPUT_FILE" ]; then
	echo "ERROR: Config input file $CONFIG_INPUT_FILE could not be found, exiting."
	exit 1
fi

mkdir -p "/host$CONTAINERD_IMPORTS_DIR"

while true; do
	if ! cmp -s "$CONFIG_INPUT_FILE" "$TARGET_FILE" ; then
		echo "applying registry mirrors"
		cp -f "$CONFIG_INPUT_FILE" "$TARGET_FILE"

		echo "restarting containerd"
		chroot /host systemctl restart containerd.service
		echo "applied registry mirrors, sleeping for a minute"
	else
		echo "no changes required, sleeping for a minute"
	fi
	sleep 60
done
`
)

var configTemplate *template.Template

func init() {
	configTemplate = template.Must(template.New("").
		Parse(`# governed by gardener-extension-registry-cache, do not edit
{{ range $mirror := . -}}
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."{{ $mirror.Host }}"]
  endpoint = ["{{ $mirror.Endpoint }}"]
{{ end -}}
`))
}

func (c *criEnsurer) Ensure() ([]client.Object, error) {
	if c.Labels == nil {
		c.Labels = map[string]string{
			"app": criEnsurerName,
		}
	}

	toml, err := c.configToml()
	if err != nil {
		return nil, fmt.Errorf("unable to template toml: %w", err)
	}

	const (
		reconcileScriptKey = "reconcile.sh"
		configTomlKey      = "70-extension-registry-cache.toml"
		workMountPath      = "/work"
	)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      criEnsurerName,
			Namespace: c.Namespace,
			Labels:    c.Labels,
		},
		Data: map[string]string{
			reconcileScriptKey: reconcileScript,
			configTomlKey:      toml,
		},
	}
	utilruntime.Must(kubernetes.MakeUnique(configMap))

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      criEnsurerName,
			Namespace: registryCacheNamespaceName,
			Labels:    c.Labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: c.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: c.Labels,
				},
				Spec: corev1.PodSpec{
					HostPID: true,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser:    pointer.Int64(0),
						RunAsGroup:   pointer.Int64(0),
						RunAsNonRoot: pointer.Bool(false),
					},
					Containers: []corev1.Container{
						{
							Name:  criEnsurerName,
							Image: c.CRIEnsurerImage.String(),
							Command: []string{
								"bash",
								"-c",
								fmt.Sprintf("%s/%s %s/%s", workMountPath, reconcileScriptKey, workMountPath, configTomlKey),
							},
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "work",
									ReadOnly:  true,
									MountPath: workMountPath,
								},
								{
									Name:      "host",
									MountPath: "/host",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "work",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMap.Name,
									},
									Items: []corev1.KeyToPath{
										{
											// only make reconcile script executable but not config file
											Key:  reconcileScriptKey,
											Path: reconcileScriptKey,
											Mode: pointer.Int32(int32(0744)),
										},
										{
											Key:  configTomlKey,
											Path: configTomlKey,
										},
									},
									Optional: pointer.Bool(false),
								},
							},
						},
						{
							Name: "host",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
	}

	return []client.Object{
		configMap,
		daemonSet,
	}, nil
}

func (c *criEnsurer) configToml() (string, error) {
	type criMirror struct {
		Host     string
		Endpoint string
	}

	var mirrors []*criMirror
	for i := range c.ReferencedServices.Items {
		svc := c.ReferencedServices.Items[i]
		mirrors = append(mirrors, &criMirror{
			Host:     svc.Labels[registryCacheServiceUpstreamLabel],
			Endpoint: fmt.Sprintf("http://%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port),
		})
	}

	var buf bytes.Buffer
	if err := configTemplate.Execute(&buf, mirrors); err != nil {
		return "", err
	}

	return buf.String(), nil
}
