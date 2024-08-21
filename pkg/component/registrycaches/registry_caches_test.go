// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registrycaches_test

import (
	"context"
	"encoding/base64"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/retry"
	retryfake "github.com/gardener/gardener/pkg/utils/retry/fake"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	fakesecretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha3"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/component/registrycaches"
)

var _ = Describe("RegistryCaches", func() {
	const (
		managedResourceName = "extension-registry-cache"

		namespace = "some-namespace"
		image     = "some-image:some-tag"
	)

	var (
		ctx        = context.Background()
		dockerSize = resource.MustParse("10Gi")
		arSize     = resource.MustParse("20Gi")
		logger     = logr.Discard()

		c                     client.Client
		shootClient           client.Client
		secretsManager        secretsmanager.Interface
		values                Values
		managedResource       *resourcesv1alpha1.ManagedResource
		managedResourceSecret *corev1.Secret

		registryCaches component.DeployWaiter
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		shootClient = fakeclient.NewClientBuilder().WithScheme(kubernetes.ShootScheme).Build()
		secretsManager = fakesecretsmanager.New(c, namespace)
		values = Values{
			Image:      image,
			VPAEnabled: true,
			Caches: []api.RegistryCache{
				{
					Upstream: "docker.io",
					Volume: &api.Volume{
						Size: &dockerSize,
					},
					GarbageCollection: &api.GarbageCollection{
						TTL: metav1.Duration{Duration: 14 * 24 * time.Hour},
					},
				},
				{
					Upstream: "europe-docker.pkg.dev",
					Volume: &api.Volume{
						Size:             &arSize,
						StorageClassName: ptr.To("premium"),
					},
					GarbageCollection: &api.GarbageCollection{
						TTL: metav1.Duration{Duration: 0},
					},
				},
			},
			ResourceReferences: []gardencorev1beta1.NamedResourceReference{},
			CacheStatuses:      []api.RegistryCacheStatus{},
			RegistryStatus:     &v1alpha3.RegistryStatus{},
		}

		managedResource = &resourcesv1alpha1.ManagedResource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedResourceName,
				Namespace: namespace,
			},
		}
		managedResourceSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedResource.Name,
				Namespace: namespace,
			},
		}
	})

	JustBeforeEach(func() {
		registryCaches = New(c, shootClient, secretsManager, logger, namespace, values)
	})

	Describe("#Deploy", func() {
		var (
			configSecretYAMLFor = func(secretName, name, upstream, configYAML string) string {
				return `apiVersion: v1
data:
  config.yml: ` + encodeBase64(configYAML) + `
immutable: true
kind: Secret
metadata:
  creationTimestamp: null
  labels:
    app: ` + name + `
    resources.gardener.cloud/garbage-collectable-reference: "true"
    upstream-host: ` + upstream + `
  name: ` + secretName + `
  namespace: kube-system
`
			}

			configYAMLFor = func(upstreamURL string, ttl string, username, password string) string {
				config := `# Maintain this file with the default config file (/etc/distribution/config.yml) from the registry image (europe-docker.pkg.dev/gardener-project/releases/3rd/registry:3.0.0-beta.1).
version: 0.1
log:
  fields:
    service: registry
storage:
  delete:
    enabled: true
  # Mitigate https://github.com/distribution/distribution/issues/2367 by disabling the blobdescriptor cache.
  # For more details, see https://github.com/distribution/distribution/issues/2367#issuecomment-1874449361.
  # cache:
  #  blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
  tag:
    concurrencylimit: 5
http:
  addr: :5000
  debug:
    addr: :5001
    prometheus:
      enabled: true
      path: /metrics
  draintimeout: 25s
  tls:
    certificate: /etc/docker/registry/certs/tls.crt
    key: /etc/docker/registry/certs/tls.key
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
proxy:
  remoteurl: ` + upstreamURL + `
  ttl: ` + ttl + `
`

				if username != "" && password != "" {
					config += `  username: ` + username + `
  password: '` + password + `'
`
				}

				return config
			}

			secretYAMLFor = func(crt, key []byte, name, upstream, secretName string) string {
				secret := `apiVersion: v1
data:
  ca.crt: ` + encodeBase64(string(crt)) + `
  ca.key: ` + encodeBase64(string(key)) + `
immutable: true
kind: Secret
metadata:
  creationTimestamp: null
  labels:
    app: ` + name + `
    resources.gardener.cloud/garbage-collectable-reference: "true"
    upstream-host: ` + upstream + `
  name: ` + secretName + `
  namespace: kube-system
type: Opaque
`
				return secret
			}

			statefulSetYAMLFor = func(name, upstream, size, configSecretName, tlsSecretName, checksum string, storageClassName *string) string {
				out := `apiVersion: apps/v1
kind: StatefulSet
metadata:
  creationTimestamp: null
  labels:
    app: ` + name + `
    upstream-host: ` + upstream + `
  name: ` + name + `
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ` + name + `
      upstream-host: ` + upstream + `
  serviceName: ` + name + `
  template:
    metadata:
      annotations:
        checksum/secret-` + name + `-tls: ` + checksum + `
      creationTimestamp: null
      labels:
        app: ` + name + `
        networking.gardener.cloud/to-dns: allowed
        networking.gardener.cloud/to-public-networks: allowed
        upstream-host: ` + upstream + `
    spec:
      automountServiceAccountToken: false
      containers:
      - env:
        - name: OTEL_TRACES_EXPORTER
          value: none
        image: ` + image + `
        imagePullPolicy: IfNotPresent
        livenessProbe:
          failureThreshold: 6
          httpGet:
            path: /debug/health
            port: 5001
          periodSeconds: 20
          successThreshold: 1
        name: registry-cache
        ports:
        - containerPort: 5000
          name: registry-cache
        - containerPort: 5001
          name: debug
        readinessProbe:
          failureThreshold: 3
          httpGet:
            path: /debug/health
            port: 5001
          periodSeconds: 20
          successThreshold: 1
        resources:
          requests:
            cpu: 20m
            memory: 50Mi
        volumeMounts:
        - mountPath: /var/lib/registry
          name: cache-volume
        - mountPath: /etc/distribution
          name: config-volume
        - mountPath: /etc/docker/registry/certs
          name: certs-volume
      priorityClassName: system-cluster-critical
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      volumes:
      - name: config-volume
        secret:
          secretName: ` + configSecretName + `
      - name: certs-volume
        secret:
          secretName: ` + tlsSecretName + `
  updateStrategy: {}
  volumeClaimTemplates:
  - metadata:
      creationTimestamp: null
      labels:
        app: ` + name + `
        upstream-host: ` + upstream + `
      name: cache-volume
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: ` + size

				if storageClassName != nil {
					out += `
      storageClassName: ` + *storageClassName
				}

				out += `
    status: {}
status:
  availableReplicas: 0
  replicas: 0
`

				return out
			}

			vpaYAMLFor = func(name string) string {
				return `apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  creationTimestamp: null
  name: ` + name + `
  namespace: kube-system
spec:
  resourcePolicy:
    containerPolicies:
    - containerName: '*'
      controlledValues: RequestsOnly
      maxAllowed:
        cpu: "4"
        memory: 8Gi
      minAllowed:
        memory: 20Mi
  targetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: ` + name + `
  updatePolicy:
    updateMode: Auto
status: {}
`
			}
		)

		Context("when cache volume size is nil", func() {
			BeforeEach(func() {
				values.Caches = []api.RegistryCache{
					{
						Upstream: "docker.io",
					}}
			})

			It("should return error", func() {
				Expect(registryCaches.Deploy(ctx)).To(MatchError(ContainSubstring("registry cache volume size is required")))
			})
		})

		Context("when VPA is enabled", func() {
			It("should successfully deploy the resources", func() {
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(registryCaches.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				expectedMr := &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:            managedResource.Name,
						Namespace:       managedResource.Namespace,
						ResourceVersion: "1",
						Labels:          map[string]string{"origin": "registry-cache"},
					},
					Spec: resourcesv1alpha1.ManagedResourceSpec{
						DeletePersistentVolumeClaims: ptr.To(true),
						InjectLabels:                 map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
						SecretRefs: []corev1.LocalObjectReference{{
							Name: managedResource.Spec.SecretRefs[0].Name,
						}},
						KeepObjects: ptr.To(false),
					},
				}
				utilruntime.Must(references.InjectAnnotations(expectedMr))
				Expect(managedResource).To(DeepEqual(expectedMr))

				managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
				Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
				Expect(managedResourceSecret.Immutable).To(Equal(ptr.To(true)))
				Expect(managedResourceSecret.Labels["resources.gardener.cloud/garbage-collectable-reference"]).To(Equal("true"))

				manifests, err := test.ExtractManifestsFromManagedResourceData(managedResourceSecret.Data)
				Expect(err).NotTo(HaveOccurred())
				Expect(manifests).To(HaveLen(8))

				dockerConfigSecretName := "registry-docker-io-config-c5a518bf"
				arConfigSecretName := "registry-europe-docker-pkg-dev-config-b666ec9c"

				dockerSecret, ok := secretsManager.Get("docker.io-tls")
				Expect(ok).To(BeTrue())
				arSecret, ok := secretsManager.Get("europe-docker.pkg.dev-tls")
				Expect(ok).To(BeTrue())
				dockerTLSSecretData := map[string][]byte{"ca.crt": dockerSecret.Data["ca.crt"], "ca.key": dockerSecret.Data["ca.key"]}
				dockerTLSSecretName := "registry-docker-io-tls-" + utils.ComputeSecretChecksum(dockerTLSSecretData)[:8]
				arTLSSecretData := map[string][]byte{"ca.crt": arSecret.Data["ca.crt"], "ca.key": arSecret.Data["ca.key"]}
				arTLSSecretName := "registry-europe-docker-pkg-dev-tls-" + utils.ComputeSecretChecksum(arTLSSecretData)[:8]

				expectedManifests := []string{
					configSecretYAMLFor(dockerConfigSecretName, "registry-docker-io", "docker.io", configYAMLFor("https://registry-1.docker.io", "336h0m0s", "", "")),
					secretYAMLFor(dockerSecret.Data["ca.crt"], dockerSecret.Data["ca.key"], "registry-docker-io", "docker.io", dockerTLSSecretName),
					statefulSetYAMLFor("registry-docker-io", "docker.io", "10Gi", dockerConfigSecretName, dockerTLSSecretName, utils.ComputeChecksum(dockerTLSSecretData), nil),
					vpaYAMLFor("registry-docker-io"),
					configSecretYAMLFor(arConfigSecretName, "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", configYAMLFor("https://europe-docker.pkg.dev", "0s", "", "")),
					secretYAMLFor(arSecret.Data["ca.crt"], arSecret.Data["ca.key"], "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", arTLSSecretName),
					statefulSetYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "20Gi", arConfigSecretName, arTLSSecretName, utils.ComputeChecksum(arTLSSecretData), ptr.To("premium")),
					vpaYAMLFor("registry-europe-docker-pkg-dev"),
				}
				Expect(manifests).To(ConsistOf(expectedManifests))
			})
		})

		Context("when VPA is disabled", func() {
			BeforeEach(func() {
				values.VPAEnabled = false
			})

			It("should successfully deploy the resources", func() {
				Expect(registryCaches.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

				manifests, err := test.ExtractManifestsFromManagedResourceData(managedResourceSecret.Data)
				Expect(err).NotTo(HaveOccurred())
				Expect(manifests).To(HaveLen(6))

				dockerConfigSecretName := "registry-docker-io-config-c5a518bf"
				arConfigSecretName := "registry-europe-docker-pkg-dev-config-b666ec9c"

				dockerSecret, ok := secretsManager.Get("docker.io-tls")
				Expect(ok).To(BeTrue())
				arSecret, ok := secretsManager.Get("europe-docker.pkg.dev-tls")
				Expect(ok).To(BeTrue())
				dockerTLSSecretData := map[string][]byte{"ca.crt": dockerSecret.Data["ca.crt"], "ca.key": dockerSecret.Data["ca.key"]}
				dockerTLSSecretName := "registry-docker-io-tls-" + utils.ComputeSecretChecksum(dockerTLSSecretData)[:8]
				arTLSSecretData := map[string][]byte{"ca.crt": arSecret.Data["ca.crt"], "ca.key": arSecret.Data["ca.key"]}
				arTLSSecretName := "registry-europe-docker-pkg-dev-tls-" + utils.ComputeSecretChecksum(arTLSSecretData)[:8]

				expectedManifests := []string{
					configSecretYAMLFor(dockerConfigSecretName, "registry-docker-io", "docker.io", configYAMLFor("https://registry-1.docker.io", "336h0m0s", "", "")),
					secretYAMLFor(dockerSecret.Data["ca.crt"], dockerSecret.Data["ca.key"], "registry-docker-io", "docker.io", dockerTLSSecretName),
					statefulSetYAMLFor("registry-docker-io", "docker.io", "10Gi", dockerConfigSecretName, dockerTLSSecretName, utils.ComputeChecksum(dockerTLSSecretData), nil),
					configSecretYAMLFor(arConfigSecretName, "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", configYAMLFor("https://europe-docker.pkg.dev", "0s", "", "")),
					secretYAMLFor(arSecret.Data["ca.crt"], arSecret.Data["ca.key"], "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", arTLSSecretName),
					statefulSetYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "20Gi", arConfigSecretName, arTLSSecretName, utils.ComputeChecksum(arTLSSecretData), ptr.To("premium")),
				}
				Expect(manifests).To(ConsistOf(expectedManifests))
			})
		})

		Context("upstream credentials are set", func() {
			var (
				dockerSecret *corev1.Secret
				arSecret     *corev1.Secret
			)

			BeforeEach(func() {
				dockerSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      "ref-docker-creds",
					},
					Data: map[string][]byte{
						"username": []byte("docker-user"),
						"password": []byte("s3cret"),
					},
				}
				arSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      "ref-ar-creds",
					},
					Data: map[string][]byte{
						"username": []byte("ar-user"),
						"password": []byte(`{"foo":"bar"}`),
					},
				}
				values.ResourceReferences = []gardencorev1beta1.NamedResourceReference{
					{Name: "docker-ref", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "docker-creds", Kind: "Secret"}},
					{Name: "ar-ref", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "ar-creds", Kind: "Secret"}},
				}
				values.Caches[0].SecretReferenceName = ptr.To("docker-ref")
				values.Caches[1].SecretReferenceName = ptr.To("ar-ref")
			})

			JustBeforeEach(func() {
				if dockerSecret != nil {
					Expect(c.Create(ctx, dockerSecret)).To(Succeed())
				}
				if arSecret != nil {
					Expect(c.Create(ctx, arSecret)).To(Succeed())
				}
			})

			It("should successfully deploy the resources", func() {
				Expect(registryCaches.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

				manifests, err := test.ExtractManifestsFromManagedResourceData(managedResourceSecret.Data)
				Expect(err).NotTo(HaveOccurred())
				Expect(manifests).To(HaveLen(8))

				dockerConfigSecretName := "registry-docker-io-config-e561062e"
				arConfigSecretName := "registry-europe-docker-pkg-dev-config-0accd8a8"

				dockerSecret, ok := secretsManager.Get("docker.io-tls")
				Expect(ok).To(BeTrue())
				arSecret, ok := secretsManager.Get("europe-docker.pkg.dev-tls")
				Expect(ok).To(BeTrue())
				dockerTLSSecretData := map[string][]byte{"ca.crt": dockerSecret.Data["ca.crt"], "ca.key": dockerSecret.Data["ca.key"]}
				dockerTLSSecretName := "registry-docker-io-tls-" + utils.ComputeSecretChecksum(dockerTLSSecretData)[:8]
				arTLSSecretData := map[string][]byte{"ca.crt": arSecret.Data["ca.crt"], "ca.key": arSecret.Data["ca.key"]}
				arTLSSecretName := "registry-europe-docker-pkg-dev-tls-" + utils.ComputeSecretChecksum(arTLSSecretData)[:8]

				expectedManifests := []string{
					configSecretYAMLFor(dockerConfigSecretName, "registry-docker-io", "docker.io", configYAMLFor("https://registry-1.docker.io", "336h0m0s", "docker-user", "s3cret")),
					secretYAMLFor(dockerSecret.Data["ca.crt"], dockerSecret.Data["ca.key"], "registry-docker-io", "docker.io", dockerTLSSecretName),
					statefulSetYAMLFor("registry-docker-io", "docker.io", "10Gi", dockerConfigSecretName, dockerTLSSecretName, utils.ComputeChecksum(dockerTLSSecretData), nil),
					vpaYAMLFor("registry-docker-io"),
					configSecretYAMLFor(arConfigSecretName, "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", configYAMLFor("https://europe-docker.pkg.dev", "0s", "ar-user", `{"foo":"bar"}`)),
					secretYAMLFor(arSecret.Data["ca.crt"], arSecret.Data["ca.key"], "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", arTLSSecretName),
					statefulSetYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "20Gi", arConfigSecretName, arTLSSecretName, utils.ComputeChecksum(arTLSSecretData), ptr.To("premium")),
					vpaYAMLFor("registry-europe-docker-pkg-dev"),
				}
				Expect(manifests).To(ConsistOf(expectedManifests))
			})

			When("get secret fails", func() {
				BeforeEach(func() {
					dockerSecret = nil
				})

				It("should return error", func() {
					err := registryCaches.Deploy(ctx)
					Expect(err).To(MatchError(ContainSubstring("failed to read referenced secret ref-docker-creds for reference docker-ref")))
				})
			})

			When("referenced resource is invalid", func() {
				BeforeEach(func() {
					values.ResourceReferences = []gardencorev1beta1.NamedResourceReference{
						{Name: "docker-ref", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "docker-creds", Kind: "ConfigMap"}},
					}
					It("should return error", func() {
						err := registryCaches.Deploy(ctx)
						Expect(err).To(MatchError(ContainSubstring("referenced resource with kind Secret not found for reference: \"docker-ref\"")))
					})
				})
			})
		})

		It("should deploy a monitoring objects", func() {
			Expect(registryCaches.Deploy(ctx)).To(Succeed())

			configMapDashboards := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "registry-cache-dashboards",
					Namespace: namespace,
				},
			}
			Expect(c.Get(ctx, client.ObjectKeyFromObject(configMapDashboards), configMapDashboards)).To(Succeed())
			Expect(configMapDashboards.Labels).To(HaveKeyWithValue("dashboard.monitoring.gardener.cloud/shoot", "true"))
			Expect(configMapDashboards.Labels).To(HaveKeyWithValue("component", "registry-cache"))
			Expect(configMapDashboards.Data).To(HaveKey("registry-cache.dashboard.json"))

			prometheusRule := &monitoringv1.PrometheusRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot-registry-cache",
					Namespace: namespace,
				},
			}
			Expect(c.Get(ctx, client.ObjectKeyFromObject(prometheusRule), prometheusRule)).To(Succeed())
			Expect(prometheusRule.Labels).To(HaveKeyWithValue("prometheus", "shoot"))
			Expect(prometheusRule.Labels).To(HaveKeyWithValue("component", "registry-cache"))
			Expect(prometheusRule.Spec.Groups[0].Name).To(Equal("registry-cache.rules"))
			Expect(prometheusRule.Spec.Groups[0].Rules).To(HaveLen(4))
			Expect(prometheusRule.Spec.Groups[0].Rules[0].Alert).To(Equal("RegistryCachePersistentVolumeUsageCritical"))
			Expect(prometheusRule.Spec.Groups[0].Rules[1].Alert).To(Equal("RegistryCachePersistentVolumeFullInFourDays"))
			Expect(prometheusRule.Spec.Groups[0].Rules[2].Record).To(Equal("shoot:registry_proxy_pushed_bytes_total:sum"))
			Expect(prometheusRule.Spec.Groups[0].Rules[3].Record).To(Equal("shoot:registry_proxy_pulled_bytes_total:sum"))

			scrapeConfig := &monitoringv1alpha1.ScrapeConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot-registry-cache",
					Namespace: namespace,
				},
			}
			Expect(c.Get(ctx, client.ObjectKeyFromObject(scrapeConfig), scrapeConfig)).To(Succeed())
			Expect(scrapeConfig.Labels).To(HaveKeyWithValue("prometheus", "shoot"))
			Expect(scrapeConfig.Labels).To(HaveKeyWithValue("component", "registry-cache"))
			Expect(scrapeConfig.Spec.Authorization.Credentials.LocalObjectReference.Name).To(Equal("shoot-access-prometheus-shoot"))
			Expect(scrapeConfig.Spec.KubernetesSDConfigs[0].APIServer).To(Equal(ptr.To("https://kube-apiserver:443")))
			Expect(scrapeConfig.Spec.RelabelConfigs).To(HaveLen(5))
			Expect(scrapeConfig.Spec.MetricRelabelConfigs).To(HaveLen(1))
			Expect(scrapeConfig.Spec.MetricRelabelConfigs[0].Regex).To(Equal("^(registry_proxy_.+)$"))
		})
	})

	Describe("#Destroy", func() {
		It("should successfully destroy all resources", func() {
			Expect(c.Create(ctx, managedResource)).To(Succeed())
			Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

			Expect(registryCaches.Destroy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))
		})
	})

	Context("waiting functions", func() {
		var fakeOps *retryfake.Ops

		BeforeEach(func() {
			fakeOps = &retryfake.Ops{MaxAttempts: 1}
			DeferCleanup(test.WithVars(
				&retry.Until, fakeOps.Until,
				&retry.UntilTimeout, fakeOps.UntilTimeout,
			))
		})

		Describe("#Wait", func() {
			It("should fail because the ManagedResource doesn't become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceName,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionFalse,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionFalse,
							},
						},
					},
				})).To(Succeed())

				Expect(registryCaches.Wait(ctx)).To(MatchError(ContainSubstring("is not healthy")))
			})

			It("should successfully wait for the managed resource to become healthy", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, &resourcesv1alpha1.ManagedResource{
					ObjectMeta: metav1.ObjectMeta{
						Name:       managedResourceName,
						Namespace:  namespace,
						Generation: 1,
					},
					Status: resourcesv1alpha1.ManagedResourceStatus{
						ObservedGeneration: 1,
						Conditions: []gardencorev1beta1.Condition{
							{
								Type:   resourcesv1alpha1.ResourcesApplied,
								Status: gardencorev1beta1.ConditionTrue,
							},
							{
								Type:   resourcesv1alpha1.ResourcesHealthy,
								Status: gardencorev1beta1.ConditionTrue,
							},
						},
					},
				})).To(Succeed())

				Expect(registryCaches.Wait(ctx)).To(Succeed())
			})
		})

		Describe("#WaitCleanup", func() {
			It("should fail when the wait for the managed resource deletion times out", func() {
				fakeOps.MaxAttempts = 2

				Expect(c.Create(ctx, managedResource)).To(Succeed())

				Expect(registryCaches.WaitCleanup(ctx)).To(MatchError(ContainSubstring("still exists")))
			})

			It("should not return an error when it's already removed", func() {
				Expect(registryCaches.WaitCleanup(ctx)).To(Succeed())
			})
		})
	})

	DescribeTable("#computeUpstreamLabel",
		func(upstream, expected string) {
			actual := ComputeUpstreamLabelValue(upstream)
			Expect(len(actual)).NotTo(BeNumerically(">", 43))
			Expect(actual).To(Equal(expected))
		},

		Entry("short upstream", "my-registry.io", "my-registry.io"),
		Entry("short upstream ends with port", "my-registry.io:5000", "my-registry.io-5000"),
		Entry("short upstream ends like a port", "my-registry.io-5000", "my-registry.io-5000"),
		Entry("long upstream", "my-very-long-registry.very-long-subdomain.io", "my-very-long-registry.very-long-subdo-2fae3"),
		Entry("long upstream ends with port", "my-very-long-registry.long-subdomain.io:8443", "my-very-long-registry.long-subdomain.-8cb9e"),
		Entry("long upstream ends like a port", "my-very-long-registry.long-subdomain.io-8443", "my-very-long-registry.long-subdomain.-e91ed"),
	)
})

func encodeBase64(val string) string {
	return base64.StdEncoding.EncodeToString([]byte(val))
}
