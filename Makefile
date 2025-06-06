# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.2.1)
# - use environment variables to overwrite this value (e.g export VERSION=0.2.1)
VERSION ?= $(shell git describe --tags --dirty --always)

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
CHANNELS="stable"
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
DEFAULT_CHANNEL="stable"
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
CONTAINER_COMMAND := $(or ${CONTAINER_COMMAND},podman)
IMAGE_BUILD_CMD ?= ${CONTAINER_COMMAND} build
IMAGE_PUSH_CMD ?= ${CONTAINER_COMMAND} push
IMAGE_BUILD_EXTRA_OPTS ?=
IMAGE_REGISTRY ?= quay.io/openshift
IMAGE_NAME := cluster-nfd-operator
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_EXTRA_TAG_NAMES ?=
IMAGE_REPO ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG ?= $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))

IMAGE_TAG_RBAC_PROXY ?= gcr.io/kubebuilder/kube-rbac-proxy:v0.8.0

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_REGISTRY)/nfd-operator-bundle:$(VERSION)

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOOS=linux
GO_CMD ?= go
GO_FMT ?= gofmt
GO=GOOS=$(GOOS) GO111MODULE=on CGO_ENABLED=0 GOFLAGS=-mod=vendor go

PACKAGE=github.com/openshift/cluster-nfd-operator
MAIN_PACKAGE=main.go
BIN=node-feature-discovery-operator
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION) -X pkg.metrics.version=$(VERSION)"

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

all: build

TESTS ?= ./...

.PHONY: unit-test
unit-test: vet ## Run tests.
	$(GO_CMD) test $(TESTS) -coverprofile cover.out

ENVTEST_ASSETS_DIR=$(PROJECT_DIR)/testbin
test-e2e: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

go_mod:
	@go mod download

# Build binary
build:
	@GOOS=$(GOOS) GO111MODULE=on CGO_ENABLED=0 $(GO_CMD) build -o $(BIN) $(LDFLAGS) $(MAIN_PACKAGE)

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

clean-manifests = (cd config/manager && $(KUSTOMIZE) edit set image controller=k8s.gcr.io/nfd/node-feature-discovery-operator:0.4.2)

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: kustomize
	cd $(PROJECT_DIR)/config/manager && \
		$(KUSTOMIZE) edit set image controller=${IMAGE_TAG}
	cd $(PROJECT_DIR)/config/default && \
		$(KUSTOMIZE) edit set image kube-rbac-proxy=${IMAGE_TAG_RBAC_PROXY}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	@$(call clean-manifests)

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy: kustomize
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=operator webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	# temp fix, remove once we stop using the downstream NodeFeatureRule, and move the upstream under api/v1alpha1
	@sed -i 's|name: v1temp1|name: v1alpha1|g' config/crd/bases/nfd.k8s-sigs.io_nodefeaturerules.yaml


# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

verify:	verify-gofmt

verify-gofmt:
ifeq (, $(GOFMT_CHECK))
	@echo "verify-gofmt: OK"
else
	@echo "verify-gofmt: ERROR: gofmt failed on the following files:"
	@echo "$(GOFMT_CHECK)"
	@echo ""
	@echo "For details, run: gofmt -d -s $(GOFMT_CHECK)"
	@echo ""
	@exit 1
endif

clean:
	go clean
	rm -f $(BIN)

# clean NFD labels on all nodes
# devel only
clean-labels:
	kubectl get no -o yaml | sed -e '/^\s*nfd.node.kubernetes.io/d' -e '/^\s*feature.node.kubernetes.io/d' | kubectl replace -f -

# Generate code
generate: controller-gen mockgen
	$(CONTROLLER_GEN) object:headerFile="utils/boilerplate.go.txt" paths="./..."
	$(GO_CMD) generate ./...

# Build the docker image
image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

# Push the docker image
push:
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag; done

site-build:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c '/usr/local/bin/bundle install && "$$BUNDLE_BIN/jekyll" build $(JEKYLL_OPTS)'

site-serve:
	@mkdir -p docs/vendor/bundle
	$(SITE_BUILD_CMD) sh -c '/usr/local/bin/bundle install && "$$BUNDLE_BIN/jekyll" serve $(JEKYLL_OPTS) -H 127.0.0.1'

.PHONY: all build test generate verify verify-gofmt clean deploy-objects deploy-operator deploy-crds push image
.SILENT: go_mod
# Download controller-gen locally if necessary
CONTROLLER_GEN = $(PROJECT_DIR)/bin/controller-gen
controller-gen:
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.1)

.PHONY: mockgen
mockgen: ## Install mockgen locally.
	$(GO_CMD) install go.uber.org/mock/mockgen@v0.5.0

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	@GOBIN=$(PROJECT_DIR)/bin  GO111MODULE=on $(GO_CMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2

# Download kustomize locally if necessary
KUSTOMIZE = $(PROJECT_DIR)/bin/kustomize
kustomize:
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.4)

# go-install-tool will 'go install' any package $2 and install it to $1.
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_TAG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle --output-dir bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	$(IMAGE_BUILD_CMD) -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# push the bundle image.
.PHONY: bundle-push
bundle-push:
	$(IMAGE_PUSH_CMD) $(BUNDLE_IMG)

# When using skippper to build bundle, uncomment the below line in order to get and use operator-sdk

#OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
#.PHONY: operator-sdk
#operator-sdk:
#	@if [ ! -f ${OPERATOR_SDK} ]; then \
#                set -e ;\
#                echo "Downloading ${OPERATOR_SDK}"; \
#                mkdir -p $(dir ${OPERATOR_SDK}) ;\
#                curl -Lo ${OPERATOR_SDK} 'https://github.com/operator-framework/operator-sdk/releases/download/v1.32.0/operator-sdk_linux_amd64'; \
#                chmod +x ${OPERATOR_SDK}; \
#        fi
