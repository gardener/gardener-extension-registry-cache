// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package extension

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
)

var _ = Describe("Actuator", func() {

	Describe("#computeProviderStatus", func() {
		It("should return a status with empty caches when no services are passed", func() {
			status := computeProviderStatus(nil, nil)

			Expect(status).To(Equal(&v1alpha3.RegistryStatus{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha3.SchemeGroupVersion.String(),
					Kind:       "RegistryStatus",
				},
				Caches:       []v1alpha3.RegistryCacheStatus{},
				CASecretName: nil,
			}))
		})

		It("should set the CASecretName when provided", func() {
			caSecretName := "ca-extension-registry-cache-1234"

			status := computeProviderStatus(nil, &caSecretName)

			Expect(status.CASecretName).To(Equal(ptr.To("ca-extension-registry-cache-1234")))
		})

		It("should compute the status for multiple services with mixed IP families", func() {
			services := []corev1.Service{
				serviceFor("10.4.246.205", "http", "docker.io", "https://registry-1.docker.io"),
				serviceFor("2a05:d018:197f:7e06::1", "https", "europe-docker.pkg.dev", "https://europe-docker.pkg.dev"),
			}
			caSecretName := "ca-extension-registry-cache-1234"

			status := computeProviderStatus(services, &caSecretName)

			Expect(status).To(Equal(&v1alpha3.RegistryStatus{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha3.SchemeGroupVersion.String(),
					Kind:       "RegistryStatus",
				},
				CASecretName: ptr.To("ca-extension-registry-cache-1234"),
				Caches: []v1alpha3.RegistryCacheStatus{
					{
						Upstream:  "docker.io",
						Endpoint:  "http://10.4.246.205:5000",
						RemoteURL: "https://registry-1.docker.io",
					},
					{
						Upstream:  "europe-docker.pkg.dev",
						Endpoint:  "https://[2a05:d018:197f:7e06::1]:5000",
						RemoteURL: "https://europe-docker.pkg.dev",
					},
				},
			}))
		})
	})
})

func serviceFor(clusterIP, scheme, upstream, remoteURL string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				constants.UpstreamAnnotation:  upstream,
				constants.SchemeAnnotation:    scheme,
				constants.RemoteURLAnnotation: remoteURL,
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIP,
		},
	}
}
