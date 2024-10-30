// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"strings"

	"github.com/gardener/gardener/pkg/utils"

	"github.com/gardener/gardener-extension-registry-cache/pkg/constants"
)

// GetUpstreamURL returns the upstream URL by given upstream.
func GetUpstreamURL(upstream string) string {
	if upstream == "docker.io" {
		return "https://registry-1.docker.io"
	}

	return "https://" + upstream
}

// GetLabels returns a map with 'app' and 'upstream-host' labels
func GetLabels(name, upstreamLabel string) map[string]string {
	return map[string]string{
		"app":                       name,
		constants.UpstreamHostLabel: upstreamLabel,
	}
}

// ComputeUpstreamLabelValue computes upstream-host label value by given upstream.
//
// Upstream is a valid DNS subdomain (RFC 1123) and optionally a port (e.g. my-registry.io[:5000])
// It is used as an 'upstream-host' label value on registry cache resources (Service, Secret, StatefulSet and VPA).
// Label values cannot contain ':' char, so if upstream is '<host>:<port>' the label value is transformed to '<host>-<port>'.
// It is also used to build the resources names escaping the '.' with '-'; e.g. `registry-<escaped_upstreamLabel>`.
//
// Due to restrictions of resource names length, if upstream length > 43 it is truncated at 37 chars, and the
// label value is transformed to <truncated-upstream>-<hash> where <hash> is first 5 chars of upstream sha256 hash.
//
// The returned upstreamLabel is at most 43 chars.
func ComputeUpstreamLabelValue(upstream string) string {
	// A label value length and a resource name length limits are 63 chars. However, Pods for a StatefulSet with name > 52 chars
	// cannot be created due to https://github.com/kubernetes/kubernetes/issues/64023.
	// The cache resources name have prefix 'registry-', thus the label value length is limited to 43.
	const labelValueLimit = 43

	upstreamLabel := strings.ReplaceAll(upstream, ":", "-")
	if len(upstream) > labelValueLimit {
		hash := utils.ComputeSHA256Hex([]byte(upstream))[:5]
		limit := labelValueLimit - len(hash) - 1
		upstreamLabel = fmt.Sprintf("%s-%s", upstreamLabel[:limit], hash)
	}
	return upstreamLabel
}
