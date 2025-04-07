// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"testing"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/gardener/gardener/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
)

func init() {
	framework.RegisterShootFrameworkFlags()
}

func TestShoot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shoot Suite")
}

// GetValidVolumeSize is to get a valid volume size.
// If the given size is smaller than the minimum volume size permitted by cloud provider on which seed cluster is running, it will return the minimum size.
//
// TODO(ialidzhikov): Remove this function in favor of github.com/gardener/gardener/pkg/apis/core/v1beta1/helper.GetValidVolumeSize
// when vendoring Gardener version that contains https://github.com/gardener/gardener/pull/11825.
func GetValidVolumeSize(seed *gardencorev1beta1.Seed, size string) string {
	if seed.Spec.Volume == nil || seed.Spec.Volume.MinimumSize == nil {
		return size
	}

	qs, err := resource.ParseQuantity(size)
	if err == nil && qs.Cmp(*seed.Spec.Volume.MinimumSize) < 0 {
		return seed.Spec.Volume.MinimumSize.String()
	}

	return size
}
