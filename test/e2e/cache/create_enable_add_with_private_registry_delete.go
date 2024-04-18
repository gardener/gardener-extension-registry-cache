package cache

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	"github.com/gardener/gardener-extension-registry-cache/test/common"
	"github.com/gardener/gardener-extension-registry-cache/test/e2e"
)

const upstreamConfigYaml = `version: 0.1
log:
  fields:
    service: registry
storage:
  inmemory:
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

var _ = Describe("Registry Cache Extension Tests", Label("cache"), func() {
	parentCtx := context.Background()

	f := e2e.DefaultShootCreationFramework()
	shoot := e2e.DefaultShoot("e2e-cache-pr")
	f.Shoot = shoot

	var (
		err           error
		pass          string
		encryptedPass []byte
		secret        *corev1.Secret
	)

	BeforeEach(func() {
		ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()

		// prepare htpasswd
		var secretBytes [32]byte
		Expect(rand.Read(secretBytes[:])).To(Equal(len(secretBytes)))
		pass = base64.RawURLEncoding.EncodeToString(secretBytes[:])

		// deploy secret in the project
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: f.ProjectNamespace,
				Name:      "ro-upstream-secret",
			},
			Immutable: ptr.To(true),
			Type:      corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte(pass),
			},
		}
		Expect(f.GardenClient.Client().Create(ctx, secret)).To(Succeed())

		encryptedPass, err = bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		Expect(err).ToNot(HaveOccurred())
		encryptedPass = append([]byte("admin:"), encryptedPass...)
	})

	AfterEach(func() {
		ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
		defer cancel()

		Expect(f.GardenClient.Client().Delete(ctx, secret)).To(Succeed())
	})

	It("should create Shoot with registry-cache extension enabled for private registry, delete Shoot", func() {
		By("Create Shoot")
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Minute)
		defer cancel()
		Expect(f.CreateShootAndWaitForCreation(ctx, false)).To(Succeed())
		f.Verify()

		By("Setup test upstream registry")
		ctx, cancel = context.WithTimeout(parentCtx, 5*time.Minute)
		defer cancel()

		// create htpasswd
		htpasswdSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metav1.NamespaceSystem,
				Name:      "test-registry-auth",
			},
			Data: map[string][]byte{
				"htpasswd": encryptedPass,
			},
		}
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, htpasswdSecret)).To(Succeed())

		// create upstream registry config
		configSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: metav1.NamespaceSystem,
				Name:      "test-registry-config",
			},
			Data: map[string][]byte{
				"config.yml": []byte(upstreamConfigYaml),
			},
		}
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, configSecret)).To(Succeed())

		// create upstream registry service
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
					Name:       "test-registry",
					Port:       5000,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("test-registry"),
				}},
				Type: corev1.ServiceTypeClusterIP,
			},
		}
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, service)).To(Succeed())

		// get service cluster IP
		Expect(f.ShootFramework.ShootClient.Client().Get(ctx, client.ObjectKeyFromObject(service), service)).To(Succeed())
		upstreamHostPort := service.Spec.ClusterIP + ":5000"

		// create upstream registry pod
		testRegistry := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-registry",
				Namespace: metav1.NamespaceSystem,
				Labels: map[string]string{
					"app": "test-registry",
				},
			},
			Spec: corev1.PodSpec{
				Hostname: "test-registry",
				Containers: []corev1.Container{
					{
						Name:  "registry",
						Image: "registry:2.8.3",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 5000,
								Name:          "test-registry",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
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
		}
		Expect(f.ShootFramework.ShootClient.Client().Create(ctx, testRegistry)).To(Succeed())
		Expect(framework.WaitUntilPodIsRunning(ctx, f.Logger, "test-registry", metav1.NamespaceSystem, f.ShootFramework.ShootClient)).To(Succeed())

		// push nginx:1.24.0 to the upstream registry
		nodeList, err := framework.GetAllNodesInWorkerPool(ctx, f.ShootFramework.ShootClient, ptr.To("local"))
		framework.ExpectNoError(err)
		rootPodExecutor := framework.NewRootPodExecutor(f.Logger, f.ShootFramework.ShootClient, &nodeList.Items[0].Name, "kube-system")

		pullImage := fmt.Sprintf("ctr content fetch --all-platforms docker.io/library/%s > /dev/null", "nginx:1.24.0")
		executeCommand(ctx, rootPodExecutor, pullImage)

		tagImage := fmt.Sprintf("ctr images tag docker.io/library/%[1]s %[2]s/%[1]s > /dev/null", "nginx:1.24.0", upstreamHostPort)
		executeCommand(ctx, rootPodExecutor, tagImage)

		pushImage := fmt.Sprintf("ctr images push --plain-http -u admin:%s %s/%s > /dev/null", pass, upstreamHostPort, "nginx:1.24.0")
		executeCommand(ctx, rootPodExecutor, pushImage)

		rmImage := fmt.Sprintf("ctr images rm %s/%s > /dev/null", upstreamHostPort, "nginx:1.24.0")
		executeCommand(ctx, rootPodExecutor, rmImage)

		rmImage = fmt.Sprintf("ctr images rm %s/%s > /dev/null", "docker.io/library", "1.24.0")
		executeCommand(ctx, rootPodExecutor, rmImage)

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
		common.VerifyRegistryCache(parentCtx, f.Logger, f.ShootFramework.ShootClient, upstreamHostPort+"/nginx:1.24.0")

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

// executeCommand executes a command on the host and checks the returned result
func executeCommand(ctx context.Context, rootPodExecutor framework.RootPodExecutor, command string) {
	response, err := rootPodExecutor.Execute(ctx, command)
	framework.ExpectNoError(err)
	Expect(response).ToNot(BeNil())
	Expect(string(response)).To(Equal(""))
}
