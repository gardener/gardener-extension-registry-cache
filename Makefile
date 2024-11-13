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
PARALLEL_E2E_TESTS          := 3
GARDENER_REPO_ROOT          ?= $(REPO_ROOT)/../gardener
SEED_NAME                   := provider-extensions
SEED_KUBECONFIG             := $(GARDENER_REPO_ROOT)/example/provider-extensions/seed/kubeconfig

ifneq ($(SEED_NAME),provider-extensions)
	SEED_KUBECONFIG := $(GARDENER_REPO_ROOT)/example/provider-extensions/seed/kubeconfig-$(SEED_NAME)
endif
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
	docker run --rm -it -v $(PWD):/go/src/github.com/gardener/gardener-extension-registry-cache golang:1.23.3 \
		sh -c "cd /go/src/github.com/gardener/gardener-extension-registry-cache \
				&& make tidy generate \
				&& chown -R $(shell id -u):$(shell id -g) ."

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg ./test

.PHONY: sast
sast: $(GOSEC)
	@bash $(GARDENER_HACK_DIR)/sast.sh

.PHONY: sast-report
sast-report: $(GOSEC)
	@bash $(GARDENER_HACK_DIR)/sast.sh --gosec-report true --report-dir $(REPO_ROOT)

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
verify: check format test sast

.PHONY: verify-extended
verify-extended: check-generate check format test-cov test-clean sast-report

test-e2e-local: $(GINKGO)
	./hack/test-e2e-local.sh --procs=$(PARALLEL_E2E_TESTS) ./test/e2e/...

ci-e2e-kind:
	./hack/ci-e2e-kind.sh

# speed-up skaffold deployments by building all images concurrently
export SKAFFOLD_BUILD_CONCURRENCY = 0
extension-up extension-dev: export SKAFFOLD_DEFAULT_REPO = garden.local.gardener.cloud:5001
extension-up extension-dev: export SKAFFOLD_PUSH = true
extension-up extension-dev: export EXTENSION_VERSION = $(VERSION)
# use static label for skaffold to prevent rolling all gardener components on every `skaffold` invocation
extension-up extension-dev extension-down: export SKAFFOLD_LABEL = skaffold.dev/run-id=extension-local

extension-up: $(SKAFFOLD) $(KIND) $(HELM) $(KUBECTL)
	@LD_FLAGS=$(LD_FLAGS) $(SKAFFOLD) run

extension-dev: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) dev --cleanup=false --trigger=manual

extension-down: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) delete
	@# The validating webhook is not part of the chart but it is created on admission Pod startup. Hence, we have to delete it explicitly.
	$(KUBECTL) delete validatingwebhookconfiguration gardener-extension-registry-cache-admission --ignore-not-found

remote-extension-up remote-extension-down: export SKAFFOLD_LABEL = skaffold.dev/run-id=extension-remote

remote-extension-up: $(SKAFFOLD) $(HELM) $(KUBECTL) $(YQ)
	@LD_FLAGS=$(LD_FLAGS) ./hack/remote-extension-up.sh --path-seed-kubeconfig $(SEED_KUBECONFIG)

remote-extension-down: $(SKAFFOLD) $(HELM) $(KUBECTL)
	$(SKAFFOLD) delete
	@# The validating webhook is not part of the chart but it is created on admission Pod startup. Hence, we have to delete it explicitly.
	$(KUBECTL) delete validatingwebhookconfiguration gardener-extension-registry-cache-admission --ignore-not-found
