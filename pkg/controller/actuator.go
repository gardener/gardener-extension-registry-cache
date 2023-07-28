// Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package controller

import (
	"context"
	"fmt"
	"time"

	extensionsconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/extension"
	"github.com/gardener/gardener/extensions/pkg/util"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/managedresources"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/config"
	"github.com/gardener/gardener-extension-registry-cache/pkg/apis/registry/v1alpha1"
	"github.com/gardener/gardener-extension-registry-cache/pkg/imagevector"
)

// NewActuator returns an actuator responsible for Extension resources.
func NewActuator(client client.Client, decoder runtime.Decoder, config config.Configuration) extension.Actuator {
	return &actuator{
		client:  client,
		decoder: decoder,
		config:  config,
	}
}

type actuator struct {
	client  client.Client
	decoder runtime.Decoder
	config  config.Configuration
}

// Reconcile the Extension resource.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	if ex.Spec.ProviderConfig == nil {
		return nil
	}

	namespace := ex.GetNamespace()

	cluster, err := controller.GetCluster(ctx, a.client, namespace)
	if err != nil {
		return err
	}

	registryConfig := &v1alpha1.RegistryConfig{}
	if _, _, err := a.decoder.Decode(ex.Spec.ProviderConfig.Raw, nil, registryConfig); err != nil {
		return fmt.Errorf("failed to decode provider config: %w", err)
	}

	return a.createResources(ctx, log, registryConfig, cluster, namespace)
}

// Delete the Extension resource.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.deleteResources(ctx, log, ex.GetNamespace())
}

// Restore the Extension resource.
func (a *actuator) Restore(ctx context.Context, log logr.Logger, ex *extensionsv1alpha1.Extension) error {
	return a.Reconcile(ctx, log, ex)
}

// Migrate the Extension resource.
func (a *actuator) Migrate(_ context.Context, _ logr.Logger, _ *extensionsv1alpha1.Extension) error {
	return nil
}

func (a *actuator) createResources(ctx context.Context, log logr.Logger, registryConfig *v1alpha1.RegistryConfig, cluster *controller.Cluster, namespace string) error {
	registryImage, err := imagevector.ImageVector().FindImage("registry")
	if err != nil {
		return fmt.Errorf("failed to find registry image: %w", err)
	}
	ensurerImage, err := imagevector.ImageVector().FindImage("cri-config-ensurer")
	if err != nil {
		return fmt.Errorf("failed to find ensurer image: %w", err)
	}

	objects := []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: registryCacheNamespaceName,
			},
		},
	}

	for _, cache := range registryConfig.Caches {
		c := registryCache{
			Namespace:                registryCacheNamespaceName,
			Upstream:                 cache.Upstream,
			VolumeSize:               *cache.Size,
			GarbageCollectionEnabled: *cache.GarbageCollectionEnabled,
			RegistryImage:            registryImage,
		}

		os, err := c.Ensure()
		if err != nil {
			return err
		}

		objects = append(objects, os...)
	}

	resources, err := managedresources.NewRegistry(kubernetes.SeedScheme, kubernetes.SeedCodec, kubernetes.SeedSerializer).AddAllAndSerialize(objects...)
	if err != nil {
		return err
	}

	// create ManagedResource for the registryCache
	err = a.createManagedResources(ctx, v1alpha1.RegistryResourceName, namespace, resources)
	if err != nil {
		return err
	}

	// get service IPs from shoot
	_, shootClient, err := util.NewClientForShoot(ctx, a.client, cluster.ObjectMeta.Name, client.Options{}, extensionsconfig.RESTOptions{})
	if err != nil {
		return fmt.Errorf("shoot client cannot be crated: %w", err)
	}

	selector := labels.NewSelector()
	r, err := labels.NewRequirement(registryCacheServiceUpstreamLabel, selection.Exists, nil)
	if err != nil {
		return err
	}
	selector = selector.Add(*r)

	// get all registry cache services
	services := &corev1.ServiceList{}
	if err := shootClient.List(ctx, services, client.InNamespace(registryCacheNamespaceName), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		log.Error(err, "could not read services from shoot")
		return err
	}

	if len(services.Items) != len(registryConfig.Caches) {
		return fmt.Errorf("not all services for all configured caches exist")
	}

	e := criEnsurer{
		Namespace:          registryCacheNamespaceName,
		CRIEnsurerImage:    ensurerImage,
		ReferencedServices: services,
	}

	os, err := e.Ensure()
	if err != nil {
		return err
	}
	objects = []client.Object{}
	objects = append(objects, os...)

	resources, err = managedresources.NewRegistry(kubernetes.SeedScheme, kubernetes.SeedCodec, kubernetes.SeedSerializer).AddAllAndSerialize(objects...)
	if err != nil {
		return err
	}

	err = a.createManagedResources(ctx, v1alpha1.RegistryEnsurerResourceName, namespace, resources)
	if err != nil {
		return err
	}

	return nil
}

func (a *actuator) deleteResources(ctx context.Context, log logr.Logger, namespace string) error {
	log.Info("Deleting managed resource for registry cache")

	if err := managedresources.Delete(ctx, a.client, namespace, v1alpha1.RegistryResourceName, false); err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return managedresources.WaitUntilDeleted(timeoutCtx, a.client, namespace, v1alpha1.RegistryResourceName)
}

func (a *actuator) createManagedResources(ctx context.Context, name, namespace string, resources map[string][]byte) error {
	var (
		injectedLabels = map[string]string{v1beta1constants.ShootNoCleanup: "true"}
		labels         = map[string]string{managedresources.LabelKeyOrigin: "registry-cache"}

		secretName, secret = managedresources.NewSecret(a.client, namespace, name, resources, false)
		// TODO(ialidzhikov): Use the managedresources.NewForShoot func when https://github.com/gardener/gardener/pull/8260
		// is adopted in the extension.
		managedResource = managedresources.New(a.client, namespace, name, "", pointer.Bool(false), labels, injectedLabels, pointer.Bool(false)).
				WithSecretRef(secretName).
				DeletePersistentVolumeClaims(true)
	)

	if err := secret.Reconcile(ctx); err != nil {
		return fmt.Errorf("could not create or update secret of managed resources: %w", err)
	}

	if err := managedResource.Reconcile(ctx); err != nil {
		return fmt.Errorf("could not create or update managed resource: %w", err)
	}

	return nil
}

func (a *actuator) updateStatus(ctx context.Context, ex *extensionsv1alpha1.Extension, _ *v1alpha1.RegistryConfig) error {
	patch := client.MergeFrom(ex.DeepCopy())
	// ex.Status.Resources = resources
	return a.client.Status().Patch(ctx, ex, patch)
}
