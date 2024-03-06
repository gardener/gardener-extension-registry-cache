# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

ENSURE_GARDENER_MOD         := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR           := $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
EXTENSION_PREFIX            := gardener-extension
NAME                        := registry-cache
ADMISSION_NAME              := $(NAME)-admission
IMAGE                       := europe-docker.pkg.dev/gardener-project/public/gardener/extensions/registry-cache
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
VERSION                     := $(shell cat "$(REPO_ROOT)/VERSION")
EFFECTIVE_VERSION           := $(VERSION)-$(shell git rev-parse HEAD)
IMAGE_TAG                   := $(EFFECTIVE_VERSION)
LD_FLAGS                    := "-w $(shell bash $(GARDENER_HACK_DIR)/get-build-ld-flags.sh k8s.io/component-base $(REPO_ROOT)/VERSION $(NAME))"
PARALLEL_E2E_TESTS          := 2

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif

#########################################
# Tools                                 #
#########################################

TOOLS_DIR := $(HACK_DIR)/tools
include $(GARDENER_HACK_DIR)/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install
install:
	@LD_FLAGS=$(LD_FLAGS) \
	bash $(GARDENER_HACK_DIR)/install.sh ./cmd/...

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

.PHONY: tidy
tidy:
	@GO111MODULE=on go mod tidy

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@bash $(GARDENER_HACK_DIR)/clean.sh ./cmd/... ./pkg/...

.PHONY: check-generate
check-generate:
	@bash $(GARDENER_HACK_DIR)/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM) $(YQ)
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check-charts.sh ./charts
	@GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) hack/check-skaffold-deps.sh

.PHONY: generate
generate: $(VGOPATH) $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(YQ)
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./charts/... ./cmd/... ./pkg/...
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) $(REPO_ROOT)/hack/update-codegen.sh

.PHONE: generate-in-docker
generate-in-docker:
	docker run --rm -it -v $(PWD):/go/src/github.com/gardener/gardener-extension-registry-cache golang:1.22.1 \
		sh -c "cd /go/src/github.com/gardener/gardener-extension-registry-cache \
				&& make tidy generate \
				&& chown -R $(shell id -u):$(shell id -g) ."

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg ./test

.PHONY: test
test:
	@bash $(GARDENER_HACK_DIR)/test.sh ./cmd/... ./pkg/...

.PHONY: test-cov
test-cov:
	@bash $(GARDENER_HACK_DIR)/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@bash $(GARDENER_HACK_DIR)/test-cover-clean.sh

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
