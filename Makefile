# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.2.1)
# - use environment variables to overwrite this value (e.g export VERSION=0.2.1)
VERSION := $(shell git describe --tags --dirty --always)

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
IMAGE_BUILD_CMD ?= podman build
IMAGE_PUSH_CMD ?= podman push
IMAGE_BUILD_EXTRA_OPTS ?=
IMAGE_REGISTRY ?= quay.io/openshift-psap
IMAGE_NAME := cluster-nfd-operator
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_EXTRA_TAG_NAMES ?=
IMAGE_REPO ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG ?= $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))

IMAGE_TAG_RBAC_PROXY ?= gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0

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
LDFLAGS= -ldflags "-s -w -X $(PACKAGE)/version.Version=$(VERSION)"

PACKAGE=github.com/openshift/cluster-nfd-operator
MAIN_PACKAGE=main.go
BIN=node-feature-discovery-operator

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
CONTROLLER_GEN = $(PROJECT_DIR)/bin/controller-gen
KUSTOMIZE = $(PROJECT_DIR)/bin/kustomize

all: build

# Run tests
ENVTEST_ASSETS_DIR=$(PROJECT_DIR)/testbin
test:
	@echo "TODO"	
go_mod:
	@$(GO_CMD) mod download

# Build binary
build: go_mod
	@GOOS=$(GOOS) GO111MODULE=on CGO_ENABLED=0 $(GO_CMD) build -o $(BIN) $(LDFLAGS) $(MAIN_PACKAGE)

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	$(GO_CMD) run ./main.go

# Install CRDs into a cluster
install: manifests 
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests 
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

clean-manifests = (cd config/manager && $(KUSTOMIZE) edit set image controller=k8s.gcr.io/nfd/node-feature-discovery-operator:0.4.2)

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: 
	cd $(PROJECT_DIR)/config/manager && \
		$(KUSTOMIZE) edit set image controller=${IMAGE_TAG}-minimal
	cd $(PROJECT_DIR)/config/default && \
		$(KUSTOMIZE) edit set image kube-rbac-proxy=${IMAGE_TAG_RBAC_PROXY}
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	@$(call clean-manifests)

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: 
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	@$(GO_FMT) -w -l $$(find . -name '*.go')

# Run go vet against code
vet:
	$(GO_CMD)  vet ./...

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

mdlint:
	find docs/ -path docs/vendor -prune -false -o -name '*.md' | xargs $(MDL) -s docs/mdl-style.rb

clean:
	$(GO_CMD)  clean
	rm -f $(BIN)

# clean NFD labels on all nodes
# devel only
clean-labels:
	kubectl get no -o yaml | sed -e '/^\s*nfd.node.kubernetes.io/d' -e '/^\s*feature.node.kubernetes.io/d' | kubectl replace -f -

# Generate code
generate: 
	$(CONTROLLER_GEN) object:headerFile="utils/boilerplate.go.txt" paths="./..."

# Build the container image
image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		$(foreach tag,$(IMAGE_EXTRA_TAGS),-t $(tag)) \
		$(IMAGE_BUILD_EXTRA_OPTS) ./

# Push the container image
push: 
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)
	for tag in $(IMAGE_EXTRA_TAGS); do $(IMAGE_PUSH_CMD) $$tag; done


# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests 
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_TAG)-minimal
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	$(IMAGE_BUILD_CMD)  -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# push the bundle image.
.PHONY: bundle-push
bundle-push:
	$(IMAGE_PUSH_CMD) $(BUNDLE_IMG)

