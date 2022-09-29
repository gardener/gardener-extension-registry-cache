package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: migrate to ginkgo test
func Test_criEnsurer_configToml(t *testing.T) {
	tests := []struct {
		name     string
		services *corev1.ServiceList
		want     string
		wantErr  bool
	}{
		{
			name: "template two services",
			services: &corev1.ServiceList{
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
			want: `# governed by gardener-extension-registry-cache, do not edit
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
  endpoint = ["http://1.1.1.1:5000"]
[plugins."io.containerd.grpc.v1.cri".registry.mirrors."ghcr.io"]
  endpoint = ["http://2.2.2.2:5001"]
`,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			c := &criEnsurer{
				ReferencedServices: tt.services,
			}
			got, err := c.configToml()
			if (err != nil) != tt.wantErr {
				t.Errorf("criEnsurer.configToml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("criEnsurer.configToml() = %v", diff)
			}
		})
	}
}
