REGISTRY       ?= quay.io
ORG            ?= zvonkok
TAG            ?= $(shell git branch | grep \* | cut -d ' ' -f2)
IMAGE          ?= ${REGISTRY}/${ORG}/cluster-nfd-operator:${TAG}
NAMESPACE      ?= openshift-nfd
PULLPOLICY     ?= IfNotPresent
TEMPLATE_CMD    = sed 's+REPLACE_IMAGE+${IMAGE}+g; s+REPLACE_NAMESPACE+${NAMESPACE}+g; s+IfNotPresent+${PULLPOLICY}+'

DEPLOY_OBJECTS  = manifests/0100_namespace.yaml manifests/0200_service_account.yaml manifests/0300_cluster_role.yaml manifests/0400_cluster_role_binding.yaml manifests/0600_operator.yaml
DEPLOY_CRDS     = manifests/0500_crd.yaml
DEPLOY_CRS      = manifests/0700_cr.yaml

PACKAGE=github.com/openshift/cluster-nfd-operator
MAIN_PACKAGE=$(PACKAGE)/cmd/manager

BIN=$(lastword $(subst /, ,$(PACKAGE)))
BINDATA=pkg/manifests/bindata.go

GOFMT_CHECK=$(shell find . -not \( \( -wholename './.*' -o -wholename '*/vendor/*' \) -prune \) -name '*.go' | sort -u | xargs gofmt -s -l)

DOCKERFILE=Dockerfile
IMAGE_TAG=openshift/origin-cluster-nfd-operator
IMAGE_REGISTRY=quay.io

vpath bin/go-bindata $(GOPATH)
GOBINDATA_BIN=bin/go-bindata

ENVVAR=GOOS=linux CGO_ENABLED=0
GOOS=linux
GO_BUILD_RECIPE=GOOS=$(GOOS) go build -mod=vendor -o $(BIN) $(MAIN_PACKAGE)

all: build

build:
	$(GO_BUILD_RECIPE)

test-e2e: 
	@${TEMPLATE_CMD} manifests/0110_namespace.yaml > manifests/operator-init.yaml
	echo -e "\n---\n" >> manifests/operator-init.yaml
	@${TEMPLATE_CMD} manifests/0200_service_account.yaml >> manifests/operator-init.yaml
	echo -e "\n---\n" >> manifests/operator-init.yaml
	@${TEMPLATE_CMD} manifests/0300_cluster_role.yaml >> manifests/operator-init.yaml
	echo -e "\n---\n" >> manifests/operator-init.yaml
	@${TEMPLATE_CMD} manifests/0600_operator.yaml >> manifests/operator-init.yaml

	go test -v ./test/e2e/... -root $(PWD) -kubeconfig=$(KUBECONFIG) -tags e2e  -globalMan manifests/0500_crd.yaml -namespacedMan manifests/operator-init.yaml 

$(DEPLOY_CRDS):
	@${TEMPLATE_CMD} $@ | kubectl apply -f -

deploy-crds: $(DEPLOY_CRDS) 
	sleep 1

deploy-objects: deploy-crds
	for obj in $(DEPLOY_OBJECTS); do \
		$(TEMPLATE_CMD) $$obj | kubectl apply -f - ;\
		sleep 1;\
	done	

deploy: deploy-objects
	@${TEMPLATE_CMD} $(DEPLOY_CRS) | kubectl apply -f -

undeploy:
	for obj in $(DEPLOY_OBJECTS) $(DEPLOY_CRDS) $(DEPLOY_CRS); do \
		$(TEMPLATE_CMD) $$obj | kubectl delete -f - ;\
	done	
	 ## Delete everything for the operator from the cluster
#	-${TEMPLATE_CMD} $(DEPLOY_OBJECTS) $(DEPLOY_OPERATOR) $(DEPLOY_CRDS) $(DEPLOY_CRS) | kubectl delete -f -

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

local-image:
	podman build --no-cache -t $(IMAGE) -f $(DOCKERFILE) .

test:
	go test ./cmd/... ./pkg/... -coverprofile cover.out

local-image-push:
	podman push $(IMAGE) 

.PHONY: all build generate verify verify-gofmt clean local-image local-image-push $(DEPLOY_OBJECTS) $(DEPLOY_OPERATOR) $(DEPLOY_CRDS) $(DEPLOY_CRS)
