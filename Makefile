# Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

EXTENSION_PREFIX            := gardener-extension
NAME                        := registry-cache
ADMISSION_NAME              := $(NAME)-admission
IMAGE                       := eu.gcr.io/gardener-project/gardener/extensions/registry-cache
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
VERSION                     := $(shell cat "$(REPO_ROOT)/VERSION")
EFFECTIVE_VERSION           := $(VERSION)-$(shell git rev-parse HEAD)
IMAGE_TAG                   := $(EFFECTIVE_VERSION)
LD_FLAGS                    := "-w $(shell $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION $(NAME))"
PARALLEL_E2E_TESTS          := 2

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif

#########################################
# Tools                                 #
#########################################

TOOLS_DIR := hack/tools
include $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install
install:
	@LD_FLAGS=$(LD_FLAGS) \
	$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install.sh ./cmd/...

.PHONY: docker-login
docker-login:
	@gcloud auth activate-service-account --key-file .kube-secrets/gcr/gcr-readwrite.json

.PHONY: docker-images
docker-images:
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(IMAGE):$(IMAGE_TAG) -f Dockerfile -m 6g --target $(NAME) .
	@docker build --build-arg EFFECTIVE_VERSION=$(EFFECTIVE_VERSION) -t $(IMAGE)-admission:$(IMAGE_TAG) -f Dockerfile -m 6g --target $(ADMISSION_NAME) .

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod tidy
	@GO111MODULE=on go mod vendor
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/.ci/*
	@$(REPO_ROOT)/hack/update-github-templates.sh
	@ln -sf ../vendor/github.com/gardener/gardener/hack/cherry-pick-pull.sh $(HACK_DIR)/cherry-pick-pull.sh

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/clean.sh ./cmd/... ./pkg/...

.PHONY: check-generate
check-generate:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM) $(YQ)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-charts.sh ./charts
	@hack/check-skaffold-deps.sh

.PHONY: generate
generate: $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(YQ)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/generate-sequential.sh ./charts/... ./cmd/... ./pkg/...
	$(MAKE) format

.PHONE: generate-in-docker
generate-in-docker:
	docker run --rm -it -v $(PWD):/go/src/github.com/gardener/gardener-extension-registry-cache golang:1.21.4 \
		sh -c "cd /go/src/github.com/gardener/gardener-extension-registry-cache \
				&& make revendor generate \
				&& chown -R $(shell id -u):$(shell id -g) ."

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg ./test

.PHONY: test
test:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh ./cmd/... ./pkg/...

.PHONY: test-cov
test-cov:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test-cov test-clean

test-e2e-local: $(GINKGO)
	./hack/test-e2e-local.sh --procs=$(PARALLEL_E2E_TESTS) ./test/e2e/...

ci-e2e-kind:
	./hack/ci-e2e-kind.sh

# use static label for skaffold to prevent rolling all gardener components on every `skaffold` invocation
extension-up extension-down: export SKAFFOLD_LABEL = skaffold.dev/run-id=extension-local

extension-up: $(SKAFFOLD) $(KIND) $(HELM)
	@LD_FLAGS=$(LD_FLAGS) $(SKAFFOLD) run

extension-dev: $(SKAFFOLD) $(HELM)
	$(SKAFFOLD) dev --cleanup=false --trigger=manual

extension-down: $(SKAFFOLD) $(HELM)
	$(SKAFFOLD) delete
