// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shoot_test

import (
	"testing"

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

// GetValidVolumeSize returns a valid volume size for the given cloud provider.
// If the given size is smaller than the minimum volume size permitted by cloud provider on which the cluster is running, it will return the minimum size.
func GetValidVolumeSize(provider string, size string) resource.Quantity {
	sizeAsQuantity := resource.MustParse(size)
	minSizeAlicloud := resource.MustParse("20Gi")

	if provider == "alicloud" && sizeAsQuantity.Cmp(minSizeAlicloud) < 0 {
		// On AliCloud the minimum size for SSD volumes is 20Gi.
		return minSizeAlicloud
	}

	return sizeAsQuantity
}
