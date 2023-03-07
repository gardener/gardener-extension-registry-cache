package controller

import (
	"fmt"

	"github.com/gardener/gardener/pkg/utils/imagevector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

var _ = Describe("criEnsurer", func() {
	var (
		ensurer *criEnsurer
		// change this when the data changes
		uniqueName = "cri-config-ensurer-ea328ef7"
		namespace  = "test"
		labels     = map[string]string{"foo": "bar"}
	)

	const reconcileScript = `
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

	BeforeEach(func() {
		ensurer = &criEnsurer{
			Namespace: namespace,
			Labels:    labels,
			CRIEnsurerImage: &imagevector.Image{
				Name:       "image",
				Repository: "repo.github.com",
				Tag:        pointer.String("v1.0"),
			},
			ReferencedServices: &corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								registryCacheServiceUpstreamLabel: "docker.io",
							},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.1.1.1",
							Ports: []corev1.ServicePort{
								{
									Port: 5000,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								registryCacheServiceUpstreamLabel: "ghcr.io",
							},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "2.2.2.2",
							Ports: []corev1.ServicePort{
								{
									Port: 5001,
								},
							},
						},
					},
				},
			},
		}
	})

	It("should create a configmap with the config file", func() {
		objects, err := ensurer.Ensure()
		Expect(err).NotTo(HaveOccurred())
		fmt.Println(objects)
		Expect(objects).To(ContainElement(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uniqueName,
				Namespace: namespace,
				Labels:    labels,
			},
			Immutable: pointer.Bool(true),
			Data: map[string]string{
				"reconcile.sh": reconcileScript,
				"70-extension-registry-cache.toml": `# governed by gardener-extension-registry-cache, do not edit
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
  endpoint = ["http://1.1.1.1:5000"]
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."ghcr.io"]
  endpoint = ["http://2.2.2.2:5001"]
`,
			},
		}))
	})
})
