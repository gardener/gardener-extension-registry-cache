// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package secrets_test

import (
	"net"
	"testing"
	"time"

	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardener-extension-registry-cache/pkg/secrets"
)

func TestSecrets(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secrets")
}

var _ = Describe("Secrets", func() {

	Describe("#ConfigsFor", func() {
		It("should return secret config for CA only when no services are passed", func() {
			services := []corev1.Service{}

			actual := secrets.ConfigsFor(services)
			Expect(actual).To(HaveLen(1))
			Expect(actual).To(ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Config": PointTo(MatchFields(IgnoreExtras, Fields{
						"Name":       Equal("ca-extension-registry-cache"),
						"CommonName": Equal("ca-extension-registry-cache"),
						"CertType":   Equal(secretutils.CACert),
						"Validity":   PointTo(Equal(730 * 24 * time.Hour)),
					})),
				}),
			))
		})

		It("should return secret configs for CA and TLS certificates", func() {
			services := []corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "registry-docker-io",
						Annotations: map[string]string{
							"upstream": "docker.io",
							"scheme":   "https",
						},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "10.4.0.10",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "registry-europe-docker-pkg-dev",
						Annotations: map[string]string{
							"upstream": "europe-docker.pkg.dev",
							"scheme":   "http",
						},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "10.4.0.11",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "registry-my-very-long-registry-very-long-subdo-2fae3",
						Annotations: map[string]string{
							"upstream": "my-very-long-registry.very-long-subdomain.io",
							"scheme":   "https",
						},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "10.4.0.12",
					},
				},
			}

			actual := secrets.ConfigsFor(services)
			Expect(actual).To(HaveLen(3))
			Expect(actual).To(ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"Config": PointTo(MatchFields(IgnoreExtras, Fields{
						"Name":       Equal("ca-extension-registry-cache"),
						"CommonName": Equal("ca-extension-registry-cache"),
						"CertType":   Equal(secretutils.CACert),
						"Validity":   PointTo(Equal(730 * 24 * time.Hour)),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"Config": PointTo(MatchFields(IgnoreExtras, Fields{
						"Name":                        Equal("registry-docker-io-tls"),
						"CommonName":                  Equal("registry-docker-io-tls"),
						"CertType":                    Equal(secretutils.ServerCert),
						"DNSNames":                    ConsistOf("registry-docker-io", "registry-docker-io.kube-system", "registry-docker-io.kube-system.svc", "registry-docker-io.kube-system.svc.cluster.local"),
						"IPAddresses":                 ConsistOf([]net.IP{net.IPv4(10, 4, 0, 10)}),
						"Validity":                    PointTo(Equal(90 * 24 * time.Hour)),
						"SkipPublishingCACertificate": BeTrue(),
					})),
				}),
				MatchFields(IgnoreExtras, Fields{
					"Config": PointTo(MatchFields(IgnoreExtras, Fields{
						"Name":       Equal("registry-my-very-long-registry-very-long-subdo-2fae3-tls"),
						"CommonName": Equal("registry-my-very-long-registry-very-long-subdo-2fae3-tls"),
						"CertType":   Equal(secretutils.ServerCert),
						"DNSNames": ConsistOf(
							"registry-my-very-long-registry-very-long-subdo-2fae3",
							"registry-my-very-long-registry-very-long-subdo-2fae3.kube-system",
							"registry-my-very-long-registry-very-long-subdo-2fae3.kube-system.svc",
							"registry-my-very-long-registry-very-long-subdo-2fae3.kube-system.svc.cluster.local",
						),
						"IPAddresses":                 ConsistOf([]net.IP{net.IPv4(10, 4, 0, 12)}),
						"Validity":                    PointTo(Equal(90 * 24 * time.Hour)),
						"SkipPublishingCACertificate": BeTrue(),
					})),
				}),
			))
		})
	})
})
