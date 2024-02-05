// Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registrycaches_test

import (
	"context"
	"encoding/base64"
	"strconv"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/resourcemanager/controller/garbagecollector/references"
	"github.com/gardener/gardener/pkg/utils/retry"
	retryfake "github.com/gardener/gardener/pkg/utils/retry/fake"
	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/gardener/gardener/pkg/utils/test/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
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

		c                     client.Client
		values                Values
		managedResource       *resourcesv1alpha1.ManagedResource
		managedResourceSecret *corev1.Secret

		registryCaches component.DeployWaiter
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		values = Values{
			Image:       image,
			VPAEnabled:  true,
			PSPDisabled: true,
			Caches: []api.RegistryCache{
				{
					Upstream: "docker.io",
					Volume: &api.Volume{
						Size: &dockerSize,
					},
					GarbageCollection: &api.GarbageCollection{
						Enabled: true,
					},
				},
				{
					Upstream: "europe-docker.pkg.dev",
					Volume: &api.Volume{
						Size:             &arSize,
						StorageClassName: pointer.String("premium"),
					},
					GarbageCollection: &api.GarbageCollection{
						Enabled: false,
					},
				},
			},
			ResourceReferences: []gardencorev1beta1.NamedResourceReference{},
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
		registryCaches = New(c, namespace, values)
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

			configYAMLFor = func(upstreamURL string, garbageCollectionEnabled bool, username, password string) string {
				config := `# Maintain this file with the default config file (/etc/docker/registry/config.yml) from the registry image (europe-docker.pkg.dev/gardener-project/releases/3rd/registry:3.0.0-alpha.1).
version: 0.1
log:
  fields:
    service: registry
storage:
  delete:
    enabled: ` + strconv.FormatBool(garbageCollectionEnabled) + `
  # Mitigate https://github.com/distribution/distribution/issues/2367 by disabling the blobdescriptor cache.
  # For more details, see https://github.com/distribution/distribution/issues/2367#issuecomment-1874449361.
  # cache:
  #  blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :5000
  debug:
    addr: :5001
    prometheus:
      enabled: true
      path: /metrics
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
proxy:
  remoteurl: ` + upstreamURL + `
`

				if username != "" && password != "" {
					config += `  username: ` + username + `
  password: '` + password + `'
`
				}

				return config
			}

			serviceYAMLFor = func(name, upstream string) string {
				return `apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    app: ` + name + `
    upstream-host: ` + upstream + `
  name: ` + name + `
  namespace: kube-system
spec:
  ports:
  - name: registry-cache
    port: 5000
    protocol: TCP
    targetPort: registry-cache
  selector:
    app: ` + name + `
    upstream-host: ` + upstream + `
  type: ClusterIP
status:
  loadBalancer: {}
`
			}

			statefulSetYAMLFor = func(name, upstream, upstreamURL, size, configSecretName, serviceAccountName string, storageClassName *string) string {
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
      creationTimestamp: null
      labels:
        app: ` + name + `
        networking.gardener.cloud/to-dns: allowed
        networking.gardener.cloud/to-public-networks: allowed
        upstream-host: ` + upstream + `
    spec:
      automountServiceAccountToken: false
      containers:
      - image: ` + image + `
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
        - mountPath: /etc/docker/registry
          name: config-volume
      priorityClassName: system-cluster-critical
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      serviceAccountName: ` + serviceAccountName + `
      volumes:
      - name: config-volume
        secret:
          secretName: ` + configSecretName + `
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

			podSecurityPolicyYAML = `apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: runtime/default
    seccomp.security.alpha.kubernetes.io/defaultProfileName: runtime/default
  creationTimestamp: null
  name: gardener.kube-system.registry-cache
spec:
  fsGroup:
    rule: RunAsAny
  runAsUser:
    rule: RunAsAny
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  volumes:
  - persistentVolumeClaim
`

			clusterRolePSPYAML = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: gardener.cloud:psp:kube-system:registry-cache
rules:
- apiGroups:
  - policy
  - extensions
  resourceNames:
  - gardener.kube-system.registry-cache
  resources:
  - podsecuritypolicies
  verbs:
  - use
`

			roleBindingPSPYAML = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  annotations:
    resources.gardener.cloud/delete-on-invalid-update: "true"
  creationTimestamp: null
  name: gardener.cloud:psp:registry-cache
  namespace: kube-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gardener.cloud:psp:kube-system:registry-cache
subjects:
- kind: ServiceAccount
  name: registry-cache
  namespace: kube-system
`

			serviceAccountYAML = `apiVersion: v1
automountServiceAccountToken: false
kind: ServiceAccount
metadata:
  creationTimestamp: null
  name: registry-cache
  namespace: kube-system
`
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

		Context("when VPA is enabled and PSP is disbaled", func() {
			It("should successfully deploy the resources", func() {
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
				Expect(registryCaches.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				expectedMr := &resourcesv1alpha1.ManagedResource{
					TypeMeta: metav1.TypeMeta{
						APIVersion: resourcesv1alpha1.SchemeGroupVersion.String(),
						Kind:       "ManagedResource",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            managedResource.Name,
						Namespace:       managedResource.Namespace,
						ResourceVersion: "1",
						Labels:          map[string]string{"origin": "registry-cache"},
					},
					Spec: resourcesv1alpha1.ManagedResourceSpec{
						DeletePersistentVolumeClaims: pointer.Bool(true),
						InjectLabels:                 map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
						SecretRefs: []corev1.LocalObjectReference{{
							Name: managedResource.Spec.SecretRefs[0].Name,
						}},
						KeepObjects: pointer.Bool(false),
					},
				}
				utilruntime.Must(references.InjectAnnotations(expectedMr))
				Expect(managedResource).To(DeepEqual(expectedMr))

				managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())
				Expect(managedResourceSecret.Type).To(Equal(corev1.SecretTypeOpaque))
				Expect(managedResourceSecret.Immutable).To(Equal(pointer.Bool(true)))
				Expect(managedResourceSecret.Labels["resources.gardener.cloud/garbage-collectable-reference"]).To(Equal("true"))

				Expect(managedResourceSecret.Data).To(HaveLen(8))
				dockerConfigSecretName := "registry-docker-io-config-c6b33c48"
				dockerConfigSecret := configSecretYAMLFor(dockerConfigSecretName, "registry-docker-io", "docker.io", configYAMLFor("https://registry-1.docker.io", true, "", ""))
				Expect(string(managedResourceSecret.Data["secret__kube-system__"+dockerConfigSecretName+".yaml"])).To(Equal(dockerConfigSecret))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-docker-io.yaml"])).To(Equal(serviceYAMLFor("registry-docker-io", "docker.io")))
				dockerStatefulSet := statefulSetYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io", "10Gi", dockerConfigSecretName, "default", nil)
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-docker-io.yaml"])).To(Equal(dockerStatefulSet))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-docker-io.yaml"])).To(Equal(vpaYAMLFor("registry-docker-io")))

				arConfigSecretName := "registry-europe-docker-pkg-dev-config-902c1c88"
				arConfigSecret := configSecretYAMLFor(arConfigSecretName, "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", configYAMLFor("https://europe-docker.pkg.dev", false, "", ""))
				Expect(string(managedResourceSecret.Data["secret__kube-system__"+arConfigSecretName+".yaml"])).To(Equal(arConfigSecret))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(serviceYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev")))
				arStatefulSet := statefulSetYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "https://europe-docker.pkg.dev", "20Gi", arConfigSecretName, "default", pointer.String("premium"))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(arStatefulSet))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(vpaYAMLFor("registry-europe-docker-pkg-dev")))
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

				Expect(managedResourceSecret.Data).To(HaveLen(6))
				Expect(managedResourceSecret.Data).ShouldNot(HaveKey(ContainSubstring("verticalpodautoscaler")))
			})
		})

		Context("PSP is not disabled", func() {
			BeforeEach(func() {
				values.PSPDisabled = false
			})

			It("should successfully deploy all resources when PSP is not disabled", func() {
				Expect(registryCaches.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

				Expect(managedResourceSecret.Data).To(HaveLen(12))
				Expect(string(managedResourceSecret.Data["serviceaccount__kube-system__registry-cache.yaml"])).To(Equal(serviceAccountYAML))
				Expect(string(managedResourceSecret.Data["podsecuritypolicy____gardener.kube-system.registry-cache.yaml"])).To(Equal(podSecurityPolicyYAML))
				Expect(string(managedResourceSecret.Data["clusterrole____gardener.cloud_psp_kube-system_registry-cache.yaml"])).To(Equal(clusterRolePSPYAML))
				Expect(string(managedResourceSecret.Data["rolebinding__kube-system__gardener.cloud_psp_registry-cache.yaml"])).To(Equal(roleBindingPSPYAML))

				dockerConfigSecretName := "registry-docker-io-config-c6b33c48"
				dockerStatefulSet := statefulSetYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io", "10Gi", dockerConfigSecretName, "registry-cache", nil)
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-docker-io.yaml"])).To(Equal(dockerStatefulSet))
				arConfigSecretName := "registry-europe-docker-pkg-dev-config-902c1c88"
				arStatefulSet := statefulSetYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "https://europe-docker.pkg.dev", "20Gi", arConfigSecretName, "registry-cache", pointer.String("premium"))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(arStatefulSet))
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
				values.Caches[0].SecretReferenceName = pointer.String("docker-ref")
				values.Caches[1].SecretReferenceName = pointer.String("ar-ref")
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

				Expect(managedResourceSecret.Data).To(HaveLen(8))
				dockerConfigSecretName := "registry-docker-io-config-239fe378"
				dockerConfigSecret := configSecretYAMLFor(dockerConfigSecretName, "registry-docker-io", "docker.io", configYAMLFor("https://registry-1.docker.io", true, "docker-user", "s3cret"))
				Expect(string(managedResourceSecret.Data["secret__kube-system__"+dockerConfigSecretName+".yaml"])).To(Equal(dockerConfigSecret))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-docker-io.yaml"])).To(Equal(serviceYAMLFor("registry-docker-io", "docker.io")))
				dockerStatefulSet := statefulSetYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io", "10Gi", dockerConfigSecretName, "default", nil)
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-docker-io.yaml"])).To(Equal(dockerStatefulSet))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-docker-io.yaml"])).To(Equal(vpaYAMLFor("registry-docker-io")))

				arConfigSecretName := "registry-europe-docker-pkg-dev-config-4bafab19"
				arConfigSecret := configSecretYAMLFor(arConfigSecretName, "registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", configYAMLFor("https://europe-docker.pkg.dev", false, "ar-user", `{"foo":"bar"}`))
				Expect(string(managedResourceSecret.Data["secret__kube-system__"+arConfigSecretName+".yaml"])).To(Equal(arConfigSecret))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(serviceYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev")))
				arStatefulSet := statefulSetYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "https://europe-docker.pkg.dev", "20Gi", arConfigSecretName, "default", pointer.String("premium"))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(arStatefulSet))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-europe-docker-pkg-dev.yaml"])).To(Equal(vpaYAMLFor("registry-europe-docker-pkg-dev")))
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

		It("should deploy a monitoring ConfigMap", func() {
			Expect(registryCaches.Deploy(ctx)).To(Succeed())

			monitoringConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extension-registry-cache-monitoring",
					Namespace: namespace,
				},
			}
			Expect(c.Get(ctx, client.ObjectKeyFromObject(monitoringConfigMap), monitoringConfigMap)).To(Succeed())
			Expect(monitoringConfigMap.Labels).To(HaveKeyWithValue("extensions.gardener.cloud/configuration", "monitoring"))
			Expect(monitoringConfigMap.Data).To(HaveKey("alerting_rules"))
			Expect(monitoringConfigMap.Data).To(HaveKey("dashboard_operators"))
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

	DescribeTable("#computeName",
		func(upstream, expected string) {
			actual := ComputeName(upstream)
			Expect(len(actual)).NotTo(BeNumerically(">", 52))
			Expect(actual).To(Equal(expected))
		},

		Entry("short upstream", "docker.io", "registry-docker-io"),
		Entry("long upstream", "myproj-releases.common.repositories.cloud.com", "registry-myproj-releases-common-repositories-c-3f834"),
	)
})

func encodeBase64(val string) string {
	return base64.StdEncoding.EncodeToString([]byte(val))
}
