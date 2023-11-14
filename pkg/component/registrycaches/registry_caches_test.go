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
	"fmt"
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

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
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
		gcrSize    = resource.MustParse("20Gi")

		c                     client.Client
		values                Values
		managedResource       *resourcesv1alpha1.ManagedResource
		managedResourceSecret *corev1.Secret

		registryCaches component.DeployWaiter
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		values = Values{
			Image:      image,
			VPAEnabled: true,
			Caches: []v1alpha1.RegistryCache{
				{
					Upstream: "docker.io",
					Size:     &dockerSize,
					GarbageCollection: &v1alpha1.GarbageCollection{
						Enabled: true,
					},
				},
				{
					Upstream: "eu.gcr.io",
					Size:     &gcrSize,
					GarbageCollection: &v1alpha1.GarbageCollection{
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

			statefulSetYAMLFor = func(name, upstream, upstreamURL, size, credentialsEnvs string, garbageCollectionEnabled bool) string {
				return `apiVersion: apps/v1
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
        upstream-host: ` + upstream + `
    spec:
      containers:
      - env:
        - name: REGISTRY_PROXY_REMOTEURL
          value: ` + upstreamURL + `
        - name: REGISTRY_STORAGE_DELETE_ENABLED
          value: "` + strconv.FormatBool(garbageCollectionEnabled) + `"
        - name: REGISTRY_HTTP_ADDR
          value: :5000
        - name: REGISTRY_HTTP_DEBUG_ADDR
          value: :5001` + credentialsEnvs + `
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
      priorityClassName: system-cluster-critical
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
          storage: ` + size + `
      storageClassName: default
    status: {}
status:
  availableReplicas: 0
  replicas: 0
`
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

			secretYAMLFor = func(name, user, password string) string {
				return `apiVersion: v1
data:
  password: ` + password + `
  username: ` + user + `
immutable: true
kind: Secret
metadata:
  creationTimestamp: null
  labels:
    resources.gardener.cloud/garbage-collectable-reference: "true"
  name: ` + name + `
  namespace: kube-system
`
			}
		)

		Context("when cache size is nil", func() {
			BeforeEach(func() {
				values.Caches = []v1alpha1.RegistryCache{
					{
						Upstream: "docker.io",
					}}
			})

			It("should return error", func() {
				Expect(registryCaches.Deploy(ctx)).To(MatchError(ContainSubstring("registry cache size is required")))
			})
		})

		Context("when VPA is enabled", func() {
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
				Expect(managedResourceSecret.Data).To(HaveLen(6))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-docker-io.yaml"])).To(Equal(serviceYAMLFor("registry-docker-io", "docker.io")))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-docker-io.yaml"])).To(Equal(statefulSetYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io", "10Gi", "", true)))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-docker-io.yaml"])).To(Equal(vpaYAMLFor("registry-docker-io")))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(serviceYAMLFor("registry-eu-gcr-io", "eu.gcr.io")))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(statefulSetYAMLFor("registry-eu-gcr-io", "eu.gcr.io", "https://eu.gcr.io", "20Gi", "", false)))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(vpaYAMLFor("registry-eu-gcr-io")))
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

				Expect(managedResourceSecret.Data).To(HaveLen(4))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-docker-io.yaml"])).To(Equal(serviceYAMLFor("registry-docker-io", "docker.io")))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-docker-io.yaml"])).To(Equal(statefulSetYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io", "10Gi", "", true)))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(serviceYAMLFor("registry-eu-gcr-io", "eu.gcr.io")))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(statefulSetYAMLFor("registry-eu-gcr-io", "eu.gcr.io", "https://eu.gcr.io", "20Gi", "", false)))
			})
		})

		Context("upstream credentials are set", func() {
			var (
				dockerSecret *corev1.Secret
				gcrSecret    *corev1.Secret

				dockerSecretChecksum = "bf30ffa0"
				gcrSecretChecksum    = "685366f9"
				dockerSecretName     = fmt.Sprintf("registry-docker-io-%s", dockerSecretChecksum)
				gcrSecretName        = fmt.Sprintf("registry-eu-gcr-io-%s", gcrSecretChecksum)

				credentialsEnvsFor = func(secret string) string {
					return `
        - name: REGISTRY_PROXY_USERNAME
          valueFrom:
            secretKeyRef:
              key: username
              name: ` + secret + `
        - name: REGISTRY_PROXY_PASSWORD
          valueFrom:
            secretKeyRef:
              key: password
              name: ` + secret
				}
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
				gcrSecret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      "ref-gcr-creds",
					},
					Data: map[string][]byte{
						"username": []byte("gcr-user"),
						"password": []byte("s3cret"),
					},
				}
				values.ResourceReferences = []gardencorev1beta1.NamedResourceReference{
					{Name: "docker-ref", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "docker-creds", Kind: "Secret"}},
					{Name: "gcr-ref", ResourceRef: autoscalingv1.CrossVersionObjectReference{Name: "gcr-creds", Kind: "Secret"}},
				}
				values.Caches[0].SecretReferenceName = pointer.String("docker-ref")
				values.Caches[1].SecretReferenceName = pointer.String("gcr-ref")
			})

			JustBeforeEach(func() {
				if dockerSecret != nil {
					Expect(c.Create(ctx, dockerSecret)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(dockerSecret), dockerSecret)).To(Succeed())
				}
				if gcrSecret != nil {
					Expect(c.Create(ctx, gcrSecret)).To(Succeed())
					Expect(c.Get(ctx, client.ObjectKeyFromObject(gcrSecret), gcrSecret)).To(Succeed())
				}
			})

			It("should successfully deploy the resources", func() {
				Expect(registryCaches.Deploy(ctx)).To(Succeed())

				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
				managedResourceSecret.Name = managedResource.Spec.SecretRefs[0].Name
				Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

				Expect(managedResourceSecret.Data).To(HaveLen(8))

				Expect(string(managedResourceSecret.Data["secret__kube-system__"+dockerSecretName+".yaml"])).To(Equal(secretYAMLFor(dockerSecretName, encodeBase64("docker-user"), encodeBase64("s3cret"))))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-docker-io.yaml"])).To(Equal(serviceYAMLFor("registry-docker-io", "docker.io")))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-docker-io.yaml"])).To(Equal(statefulSetYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io", "10Gi", credentialsEnvsFor(dockerSecretName), true)))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-docker-io.yaml"])).To(Equal(vpaYAMLFor("registry-docker-io")))

				Expect(string(managedResourceSecret.Data["secret__kube-system__"+gcrSecretName+".yaml"])).To(Equal(secretYAMLFor(gcrSecretName, encodeBase64("gcr-user"), encodeBase64("s3cret"))))
				Expect(string(managedResourceSecret.Data["service__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(serviceYAMLFor("registry-eu-gcr-io", "eu.gcr.io")))
				Expect(string(managedResourceSecret.Data["statefulset__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(statefulSetYAMLFor("registry-eu-gcr-io", "eu.gcr.io", "https://eu.gcr.io", "20Gi", credentialsEnvsFor(gcrSecretName), false)))
				Expect(string(managedResourceSecret.Data["verticalpodautoscaler__kube-system__registry-eu-gcr-io.yaml"])).To(Equal(vpaYAMLFor("registry-eu-gcr-io")))
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
})

func encodeBase64(val string) string {
	return base64.StdEncoding.EncodeToString([]byte(val))
}
