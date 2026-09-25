// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/utils"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

const (
	alpine3188Image           = "alpine:3.18.8"
	registryImage             = "europe-docker.pkg.dev/gardener-project/releases/3rd/registry:3.1.2@sha256:c87f33837722a100572e95d7dc4bf539fc42cf68202b13c3bc03c0ff54c3a649"
	upstreamRegistryNamespace = "test-registry"
	upstreamConfigYAML        = `version: 0.1
log:
  fields:
    service: registry
storage:
  filesystem:
    rootdirectory: /var/lib/registry
auth:
  htpasswd:
    realm: basic-realm
    path: /var/lib/password/htpasswd
http:
  addr: :5000
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
`
)

var _ = Describe("Registry Cache Extension Tests", Label("cache"), Ordered, func() {
	f := e2e.DefaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("e2e-cache-pr")

	var (
		password         string
		secret           *corev1.Secret
		upstreamHostPort string
	)

	It("should create Shoot", func(ctx SpecContext) {
		var err error
		password, err = utils.GenerateRandomString(32)
		Expect(err).NotTo(HaveOccurred())
		Expect(password).To(HaveLen(32))

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ro-upstream-secret",
				Namespace: f.ProjectNamespace,
			},
			Immutable: new(true),
			Type:      corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte(password),
			},
		}
		Expect(f.GardenClient.Client().Create(ctx, secret)).To(Succeed())

		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()
	}, SpecTimeout(15*time.Minute))

	It("should deploy test upstream registry", func(ctx SpecContext) {
		upstreamHostPort = deployUpstreamRegistry(ctx, f, password)
	}, SpecTimeout(3*time.Minute))

	It("should push image to the test upstream registry", func(ctx SpecContext) {
		pushImageToUpstreamRegistry(ctx, f, upstreamHostPort, password)
	}, SpecTimeout(2*time.Minute))

	It("should enable the registry-cache extension", func(ctx SpecContext) {
		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			addPrivateRegistrySecret(shoot)
			size := resource.MustParse("1Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{
					Upstream:            upstreamHostPort,
					RemoteURL:           new("http://" + upstreamHostPort),
					Volume:              &v1alpha3.Volume{Size: &size},
					SecretReferenceName: new("upstream-secret"),
				},
			})

			return nil
		})).To(Succeed())
	}, SpecTimeout(10*time.Minute))

	It("should verify registry-cache works", func(ctx SpecContext) {
		common.VerifyRegistryCache(ctx, f.Logger, f.ShootFramework.ShootClient, fmt.Sprintf("%s/%s", upstreamHostPort, alpine3188Image), common.AlpinePodMutateFn)
	}, SpecTimeout(12*time.Minute))

	It("should delete Shoot", func(ctx SpecContext) {
		namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: upstreamRegistryNamespace}}
		Expect(f.ShootFramework.ShootClient.Client().Delete(ctx, namespace)).To(Or(Succeed(), BeNotFoundError()))
		Expect(f.WaitUntilNamespaceIsDeleted(ctx, f.ShootFramework.ShootClient, upstreamRegistryNamespace)).To(Succeed())

		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())

		Expect(f.GardenClient.Client().Delete(ctx, secret)).To(Or(Succeed(), BeNotFoundError()))
	}, SpecTimeout(15*time.Minute))
})

func addPrivateRegistrySecret(shoot *gardencorev1beta1.Shoot) {
	shoot.Spec.Resources = append(shoot.Spec.Resources, gardencorev1beta1.NamedResourceReference{
		Name: "upstream-secret",
		ResourceRef: autoscalingv1.CrossVersionObjectReference{
			APIVersion: "v1",
			Kind:       "Secret",
			Name:       "ro-upstream-secret",
		},
	})
}

// deployUpstreamRegistry deploy test upstream registry and return the <host:port> to it
func deployUpstreamRegistry(ctx context.Context, f *framework.ShootCreationFramework, password string) (upstreamHostPort string) {
	// Create dedicated namespace for the upstream registry
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: upstreamRegistryNamespace}}
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Create(ctx, namespace)).To(Succeed())

	// Create htpasswd Secret
	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	encryptedPassword = append([]byte("admin:"), encryptedPassword...)

	htpasswdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry-auth",
			Namespace: upstreamRegistryNamespace,
		},
		Data: map[string][]byte{
			"htpasswd": encryptedPassword,
		},
	}
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Create(ctx, htpasswdSecret)).To(Succeed())

	// Create upstream registry config Secret
	configSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry-config",
			Namespace: upstreamRegistryNamespace,
		},
		Data: map[string][]byte{
			"config.yml": []byte(upstreamConfigYAML),
		},
	}
	Expect(f.ShootFramework.ShootClient.Client().Create(ctx, configSecret)).To(Succeed())

	// Create upstream registry Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: upstreamRegistryNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test-registry",
			},
			Ports: []corev1.ServicePort{{
				Port:     5000,
				Protocol: corev1.ProtocolTCP,
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Create(ctx, service)).To(Succeed())

	// Get Service's cluster IP
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Get(ctx, client.ObjectKeyFromObject(service), service)).To(Succeed())
	upstreamHostPort = service.Spec.ClusterIP + ":5000"

	// Create upstream registry StatefulSet
	registry := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: upstreamRegistryNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: service.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-registry",
				},
			},
			Replicas: new(int32(1)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-registry",
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: new(false),
					PriorityClassName:            "system-cluster-critical",
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "registry",
							Image:           registryImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5000,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "test-registry-store",
									ReadOnly:  false,
									MountPath: "/var/lib/registry",
								},
								{
									Name:      "htpasswd-volume",
									MountPath: "/var/lib/password/htpasswd",
									SubPath:   "htpasswd",
								},
								{
									Name:      "config-volume",
									MountPath: "/etc/distribution",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: configSecret.Name,
								},
							},
						},
						{
							Name: "htpasswd-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: htpasswdSecret.Name,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-registry-store",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
		},
	}
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Create(ctx, registry)).To(Succeed())
	ExpectWithOffset(1, f.WaitUntilStatefulSetIsRunning(ctx, "test-registry", upstreamRegistryNamespace, f.ShootFramework.ShootClient)).To(Succeed())

	// Alow traffic to test registry
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-test-registry",
			Namespace: upstreamRegistryNamespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "test-registry"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: new(corev1.ProtocolTCP), Port: new(intstr.FromInt32(5000))}},
			}},
		},
	}
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Create(ctx, networkPolicy)).To(Succeed())

	return
}

// pushImageToUpstreamRegistry pushes the alpine:3.18.8 image to the upstream registry.
func pushImageToUpstreamRegistry(ctx context.Context, f *framework.ShootCreationFramework, upstreamHostPort, password string) {
	nodeList, err := framework.GetAllNodesInWorkerPool(ctx, f.ShootFramework.ShootClient, new("local"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nodeList.Items).ToNot(BeEmpty(), "Expected to find at least one Node in the cluster")

	rootPodExecutor := framework.NewRootPodExecutor(f.Logger, f.ShootFramework.ShootClient, &nodeList.Items[0].Name, metav1.NamespaceSystem)
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images pull --all-platforms %s > /dev/null", common.GithubRegistryJitesoftAlpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images tag %s %s/%s > /dev/null", common.GithubRegistryJitesoftAlpine3188Image, upstreamHostPort, alpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images push --plain-http -u admin:%s %s/%s > /dev/null", password, upstreamHostPort, alpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images rm %s/%s > /dev/null", upstreamHostPort, alpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images rm %s > /dev/null", common.GithubRegistryJitesoftAlpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	ExpectWithOffset(1, rootPodExecutor.Clean(ctx)).To(Succeed())
}
