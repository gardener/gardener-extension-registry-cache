// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registrycacheservices_test

import (
	"context"
	"testing"

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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry"
	. "github.com/gardener/gardener-extension-registry-cache/pkg/component/registrycacheservices"
)

func TestRegistryCacheServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component RegistryCacheServices Suite")
}

var _ = Describe("RegistryCaches", func() {
	const (
		managedResourceName = "extension-registry-cache-services"

		namespace = "some-namespace"
	)

	var (
		ctx = context.Background()

		c                     client.Client
		values                Values
		managedResource       *resourcesv1alpha1.ManagedResource
		managedResourceSecret *corev1.Secret

		registryCacheServices component.DeployWaiter
	)

	BeforeEach(func() {
		c = fakeclient.NewClientBuilder().WithScheme(kubernetes.SeedScheme).Build()
		values = Values{
			Caches: []api.RegistryCache{
				{
					Upstream: "docker.io",
				},
				{
					Upstream: "europe-docker.pkg.dev",
				},
			},
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
		registryCacheServices = New(c, namespace, values)
	})

	Describe("#Deploy", func() {
		var serviceYAMLFor = func(name, upstream, remoteURL string) string {
			return `apiVersion: v1
kind: Service
metadata:
  annotations:
    remote-url: ` + remoteURL + `
    upstream: ` + upstream + `
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

		It("should successfully deploy the resources", func() {
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
			Expect(registryCacheServices.Deploy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
			expectedMr := &resourcesv1alpha1.ManagedResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:            managedResource.Name,
					Namespace:       managedResource.Namespace,
					ResourceVersion: "1",
					Labels:          map[string]string{"origin": "registry-cache"},
				},
				Spec: resourcesv1alpha1.ManagedResourceSpec{
					InjectLabels: map[string]string{"shoot.gardener.cloud/no-cleanup": "true"},
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
			Expect(manifests).To(HaveLen(2))

			expectedManifests := []string{
				serviceYAMLFor("registry-docker-io", "docker.io", "https://registry-1.docker.io"),
				serviceYAMLFor("registry-europe-docker-pkg-dev", "europe-docker.pkg.dev", "https://europe-docker.pkg.dev"),
			}
			Expect(manifests).To(ConsistOf(expectedManifests))
			Expect(manifests).To(ConsistOf(expectedManifests))
		})
	})

	Describe("#Delete", func() {
		It("should successfully destroy all resources", func() {
			Expect(c.Create(ctx, managedResource)).To(Succeed())
			Expect(c.Create(ctx, managedResourceSecret)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(Succeed())
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(Succeed())

			Expect(registryCacheServices.Destroy(ctx)).To(Succeed())

			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResource), managedResource)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: resourcesv1alpha1.SchemeGroupVersion.Group, Resource: "managedresources"}, managedResource.Name)))
			Expect(c.Get(ctx, client.ObjectKeyFromObject(managedResourceSecret), managedResourceSecret)).To(MatchError(apierrors.NewNotFound(schema.GroupResource{Group: corev1.SchemeGroupVersion.Group, Resource: "secrets"}, managedResourceSecret.Name)))
		})
	})

	Context("waiting functions", func() {
		var fakeOps *retryfake.Ops

		BeforeEach(func() {
			fakeOps = &retryfake.Ops{MaxAttempts: 2}
			DeferCleanup(test.WithVars(
				&retry.Until, fakeOps.Until,
				&retry.UntilTimeout, fakeOps.UntilTimeout,
			))
		})

		Describe("#Wait", func() {
			It("should fail because the ManagedResource doesn't become healthy", func() {
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

				Expect(registryCacheServices.Wait(ctx)).To(MatchError(ContainSubstring("is not healthy")))
			})

			It("should successfully wait for the managed resource to become healthy", func() {
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

				Expect(registryCacheServices.Wait(ctx)).To(Succeed())
			})
		})

		Describe("#WaitCleanup", func() {
			It("should fail when the wait for the managed resource deletion times out", func() {
				Expect(c.Create(ctx, managedResource)).To(Succeed())

				Expect(registryCacheServices.WaitCleanup(ctx)).To(MatchError(ContainSubstring("still exists")))
			})

			It("should not return an error when it's already removed", func() {
				Expect(registryCacheServices.WaitCleanup(ctx)).To(Succeed())
			})
		})
	})
})
