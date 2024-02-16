// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package registry

// GetUpstreamURL returns the upstream URL by given upstream.
func GetUpstreamURL(upstream string) string {
	if upstream == "docker.io" {
		return "https://registry-1.docker.io"
	}

	return "https://" + upstream
}
