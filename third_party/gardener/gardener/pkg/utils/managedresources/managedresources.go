// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package managedresources

import (
	"context"
	"fmt"
	"time"

	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/kubernetes/health"
	"github.com/gardener/gardener/pkg/utils/retry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IntervalWait is the interval when waiting for managed resources.
var IntervalWait = 2 * time.Second

// WaitUntilHealthy waits until the given managed resource is healthy.
func WaitUntilHealthy(ctx context.Context, client client.Reader, namespace, name string) error {
	return waitUntilHealthy(ctx, client, namespace, name, false)
}

func waitUntilHealthy(ctx context.Context, c client.Reader, namespace, name string, andNotProgressing bool) error {
	obj := &resourcesv1alpha1.ManagedResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return retry.Until(ctx, IntervalWait, func(ctx context.Context) (done bool, err error) {
		if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, obj); err != nil {
			return retry.SevereError(err)
		}

		if err := health.CheckManagedResource(obj); err != nil {
			return retry.MinorError(fmt.Errorf("managed resource %s/%s is not healthy", namespace, name))
		}

		if andNotProgressing {
			if err := health.CheckManagedResourceProgressing(obj); err != nil {
				return retry.MinorError(fmt.Errorf("managed resource %s/%s is still progressing", namespace, name))
			}
		}

		return retry.Ok()
	})
}
