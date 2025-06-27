// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/utils"
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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

const (
	alpine3188         = "alpine:3.18.8"
	registry300Image   = "europe-docker.pkg.dev/gardener-project/releases/3rd/registry:3.0.0"
	upstreamConfigYAML = `version: 0.1
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

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	parentCtx := context.Background()

	f := e2e.DefaultShootCreationFramework()
	f.Shoot = e2e.DefaultShoot("e2e-cache-pr")

	var (
		password string
		secret   *corev1.Secret
	)

	BeforeEach(func() {
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()

		// Prepare htpasswd
		var err error
		password, err = utils.GenerateRandomString(32)
		Expect(err).NotTo(HaveOccurred())
		Expect(password).To(HaveLen(32))

		// Create Secret in the Project namespace
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ro-upstream-secret",
				Namespace: f.ProjectNamespace,
			},
			Immutable: ptr.To(true),
			Type:      corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte(password),
			},
		}
		Expect(f.GardenClient.Client().Create(ctx, secret)).To(Succeed())
	})

	AfterEach(func() {
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
		defer cancel()

		Expect(f.GardenClient.Client().Delete(ctx, secret)).To(Succeed())
	})

	It("should create Shoot, enable extension for private registry, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Deploy test upstream registry")
		ctx, cancel = context.WithTimeout(parentCtx, 3*time.Minute)
		defer cancel()
		upstreamHostPort := deployUpstreamRegistry(ctx, f, password)

		By("Push image to the test upstream registry")
		ctx, cancel = context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()
		pushImageToUpstreamRegistry(ctx, f, upstreamHostPort, password)

		By("Enable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()

		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			addPrivateRegistrySecret(shoot)
			size := resource.MustParse("1Gi")
			common.AddOrUpdateRegistryCacheExtension(shoot, []v1alpha3.RegistryCache{
				{
					Upstream:            upstreamHostPort,
					RemoteURL:           ptr.To("http://" + upstreamHostPort),
					Volume:              &v1alpha3.Volume{Size: &size},
					SecretReferenceName: ptr.To("upstream-secret"),
				},
			})

			return nil
		})).To(Succeed())

		By("[" + upstreamHostPort + "] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, fmt.Sprintf("%s/%s", upstreamHostPort, alpine3188), common.AlpinePodMutateFn)

		By("Delete Shoot")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()
		Expect(f.DeleteShootAndWaitForDeletion(ctx, f.Shoot)).To(Succeed())
	})
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
	// Create htpasswd Secret
	encryptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	encryptedPassword = append([]byte("admin:"), encryptedPassword...)

	htpasswdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry-auth",
			Namespace: metav1.NamespaceSystem,
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
			Namespace: metav1.NamespaceSystem,
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
			Namespace: metav1.NamespaceSystem,
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
			Namespace: metav1.NamespaceSystem,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: service.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test-registry",
				},
			},
			Replicas: ptr.To[int32](1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-registry",
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr.To(false),
					PriorityClassName:            "system-cluster-critical",
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "registry",
							Image:           registry300Image,
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
	ExpectWithOffset(1, f.WaitUntilStatefulSetIsRunning(ctx, "test-registry", metav1.NamespaceSystem, f.ShootFramework.ShootClient)).To(Succeed())

	// Alow traffic to test registry
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-test-registry",
			Namespace: metav1.NamespaceSystem,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "test-registry"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				Ports: []networkingv1.NetworkPolicyPort{{Protocol: ptr.To(corev1.ProtocolTCP), Port: ptr.To(intstr.FromInt32(5000))}},
			}},
		},
	}
	ExpectWithOffset(1, f.ShootFramework.ShootClient.Client().Create(ctx, networkPolicy)).To(Succeed())

	return
}

// pushImageToUpstreamRegistry pushes the alpine:3.18.8 image to the upstream registry.
func pushImageToUpstreamRegistry(ctx context.Context, f *framework.ShootCreationFramework, upstreamHostPort, password string) {
	nodeList, err := framework.GetAllNodesInWorkerPool(ctx, f.ShootFramework.ShootClient, ptr.To("local"))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, nodeList.Items).ToNot(BeEmpty(), "Expected to find at least one Node in the cluster")

	rootPodExecutor := framework.NewRootPodExecutor(f.Logger, f.ShootFramework.ShootClient, &nodeList.Items[0].Name, metav1.NamespaceSystem)
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images pull --all-platforms %s > /dev/null", common.GithubRegistryJitesoftAlpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images tag %s %s/%s > /dev/null", common.GithubRegistryJitesoftAlpine3188Image, upstreamHostPort, alpine3188))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images push --plain-http -u admin:%s %s/%s > /dev/null", password, upstreamHostPort, alpine3188))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images rm %s/%s > /dev/null", upstreamHostPort, alpine3188))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = rootPodExecutor.Execute(ctx, "sh", "-c", fmt.Sprintf("ctr images rm %s > /dev/null", common.GithubRegistryJitesoftAlpine3188Image))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	ExpectWithOffset(1, rootPodExecutor.Clean(ctx)).To(Succeed())
}
