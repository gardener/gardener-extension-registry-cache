// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

const (
	//nginx1240Digest is the nginx:1.24.0 digest for multi-platform index
	nginx1240Digest    = "nginx@sha256:f6daac2445b0ce70e64d77442ccf62839f3f1b4c24bf6746a857eff014e798c8"
	indexMediaType     = "application/vnd.oci.image.config.v1+json"
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
		password          string
		encryptedPassword []byte
		secret            *corev1.Secret
	)

	BeforeEach(func() {
		ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()

		// Prepare htpasswd
		var err error
		password, err = utils.GenerateRandomString(32)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(password)).To(Equal(32))

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

		encryptedPassword, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		Expect(err).ToNot(HaveOccurred())
		encryptedPassword = append([]byte("admin:"), encryptedPassword...)
	})

	AfterEach(func() {
		ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()

		Expect(f.GardenClient.Client().Delete(ctx, secret)).To(Succeed())
	})

	It("should create Shoot, enable extension for private registry, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Setup test upstream registry")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()

		// Create htpasswd Secret
		htpasswdSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-registry-auth",
				Namespace: metav1.NamespaceSystem,
			},
			Data: map[string][]byte{
				"htpasswd": encryptedPassword,
			},
		}
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, htpasswdSecret)).To(Succeed())

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
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, service)).To(Succeed())

		// Get Service's cluster IP
		Expect(f.ShootFramework.ShootClient.Client().Get(ctx, client.ObjectKeyFromObject(service), service)).To(Succeed())
		upstreamHostPort := service.Spec.ClusterIP + ":5000"

		// Create PersistentVolume
		testRegistryPV := &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-registry-store",
			},
			Spec: corev1.PersistentVolumeSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				StorageClassName: "manual",
				Capacity: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/test-registry-store",
					},
				},
			},
		}
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, testRegistryPV)).To(Succeed())

		// Create upstream registry StatefulSet
		testRegistry := &appsv1.StatefulSet{
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
				Replicas: ptr.To(int32(1)),
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
								Image:           "registry:2.8.3",
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
										MountPath: "/etc/docker/registry",
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
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							StorageClassName: ptr.To("manual"),
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
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, testRegistry)).To(Succeed())
		Expect(f.WaitUntilStatefulSetIsRunning(ctx, "test-registry", metav1.NamespaceSystem, f.ShootFramework.ShootClient)).To(Succeed())

		// Push nginx:1.24.0 to the upstream registry
		nodeList, err := framework.GetAllNodesInWorkerPool(ctx, f.ShootFramework.ShootClient, ptr.To("local"))
		framework.ExpectNoError(err)
		rootPodExecutor := framework.NewRootPodExecutor(f.Logger, f.ShootFramework.ShootClient, &nodeList.Items[0].Name, metav1.NamespaceSystem)

		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr images pull --platform amd64 --platform arm64 docker.io/library/%s > /dev/null", nginx1240Digest))
		framework.ExpectNoError(err)
		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr images tag docker.io/library/%[1]s %[2]s/%[1]s > /dev/null", nginx1240Digest, upstreamHostPort))
		framework.ExpectNoError(err)
		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr content push-object --plain-http -u admin:%s %s/%s %s %s > /dev/null", password, upstreamHostPort, nginx1240Digest, nginx1240Digest[len("nginx@"):], indexMediaType))
		framework.ExpectNoError(err)
		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr images push --platform amd64 --plain-http -u admin:%s %s/%s > /dev/null", password, upstreamHostPort, nginx1240Digest))
		framework.ExpectNoError(err)
		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr images push --platform arm64 --plain-http -u admin:%s %s/%s > /dev/null", password, upstreamHostPort, nginx1240Digest))
		framework.ExpectNoError(err)
		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr images rm %s/%s > /dev/null", upstreamHostPort, nginx1240Digest))
		framework.ExpectNoError(err)
		_, err = rootPodExecutor.Execute(ctx, fmt.Sprintf("ctr images rm %s/%s > /dev/null", "docker.io/library", nginx1240Digest))
		framework.ExpectNoError(err)

		Expect(rootPodExecutor.Clean(ctx)).To(Succeed())

		By("Enable the registry-cache extension")
		ctx, cancel = context.WithTimeout(parentCtx, 10*time.Minute)
		defer cancel()

		Expect(f.UpdateShoot(ctx, f.Shoot, func(shoot *gardencorev1beta1.Shoot) error {
			addPrivateRegistrySecret(shoot)
			size := resource.MustParse("2Gi")
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

		By("Wait until the registry configuration is applied")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()
		common.WaitUntilRegistryCacheConfigurationsAreApplied(ctx, f.Logger, f.ShootFramework.ShootClient)

		By("[" + upstreamHostPort + "] Verify registry-cache works")
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, fmt.Sprintf("%s/%s", upstreamHostPort, "1.24.0"))

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
