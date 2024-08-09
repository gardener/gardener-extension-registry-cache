# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -e

repository=$(echo $SKAFFOLD_IMAGE | rev | cut -d':' -f 2- | rev)
tag=$(echo $SKAFFOLD_IMAGE | rev | cut -d':' -f 1 | rev)

cat <<EOF > local-setup/patch-controller-deployment-image.yaml
apiVersion: core.gardener.cloud/v1
kind: ControllerDeployment
metadata:
  name: extension-registry-cache
helm:
  values:
    image:
      repository: ${repository}
      tag: ${tag}
EOF
