# Set the shell to bash as a workaround for testing tools download failure
# Refer: https://github.com/operator-framework/operator-sdk/issues/4203.
SHELL := /bin/bash

# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= storageos/operator:test
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

API_MANAGER_IMG ?= storageos/api-manager:v1.1.2
EXTERNAL_PROVISIONER_IMG ?= storageos/csi-provisioner:v2.1.1-patched
EXTERNAL_ATTACHER_IMG ?= quay.io/k8scsi/csi-attacher:v3.1.0
EXTERNAL_RESIZER_IMG ?= quay.io/k8scsi/csi-resizer:v1.1.0
INIT_IMG ?= storageos/init:v2.1.0
NODE_IMG ?= storageos/node:v2.4.0
NODE_DRIVER_REG_IMG ?= quay.io/k8scsi/csi-node-driver-registrar:v2.1.0
LIVENESS_PROBE_IMG ?= quay.io/k8scsi/livenessprobe:v2.2.0

# The related image environment variables. These are used in the opreator's
# configuration by converting into a ConfigMap and loading as a container's
# environment variables. See make target cofig-update.
define REL_IMG_CONF
RELATED_IMAGE_API_MANAGER=${API_MANAGER_IMG}
RELATED_IMAGE_CSIV1_EXTERNAL_PROVISIONER=${EXTERNAL_PROVISIONER_IMG}
RELATED_IMAGE_CSIV1_EXTERNAL_ATTACHER_V3=${EXTERNAL_ATTACHER_IMG}
RELATED_IMAGE_CSIV1_EXTERNAL_RESIZER=${EXTERNAL_RESIZER_IMG}
RELATED_IMAGE_STORAGEOS_INIT=${INIT_IMG}
RELATED_IMAGE_STORAGEOS_NODE=${NODE_IMG}
RELATED_IMAGE_CSIV1_NODE_DRIVER_REGISTRAR=${NODE_DRIVER_REG_IMG}
RELATED_IMAGE_CSIV1_LIVENESS_PROBE=${LIVENESS_PROBE_IMG}
endef
export REL_IMG_CONF

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

e2e:
	kubectl-kuttl test --config tests/e2e/kuttl-test.yaml

# Build manager binary
manager: generate fmt vet
	CGO_ENABLED=0 go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	RELATED_IMAGE_API_MANAGER=${API_MANAGER_IMG} \
	RELATED_IMAGE_CSIV1_EXTERNAL_PROVISIONER=${EXTERNAL_PROVISIONER_IMG} \
	RELATED_IMAGE_CSIV1_EXTERNAL_ATTACHER_V3=${EXTERNAL_ATTACHER_IMG} \
	RELATED_IMAGE_CSIV1_EXTERNAL_RESIZER=${EXTERNAL_RESIZER_IMG} \
	RELATED_IMAGE_STORAGEOS_INIT=${INIT_IMG} \
	RELATED_IMAGE_STORAGEOS_NODE=${NODE_IMG} \
	RELATED_IMAGE_CSIV1_NODE_DRIVER_REGISTRAR=${NODE_DRIVER_REG_IMG} \
	RELATED_IMAGE_CSIV1_LIVENESS_PROBE=${LIVENESS_PROBE_IMG} \
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen config-update
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Update the operator configuration.
config-update:
	@echo "$$REL_IMG_CONF" > config/manager/related_image_config.yaml

install-manifest:
	$(KUSTOMIZE) build config/default > storageos-operator.yaml

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
# NOTE: The Dockerfile is written for use with goreleaser. For the same
# Dockerfile to work directly with docker, the binary is copied to the PWD and
# removed after use.
docker-build: manager
	@cp bin/manager manager
	docker build -t ${IMG} .
	@rm -f manager

# Push the docker image
docker-push:
	docker push ${IMG}

# Build development binaries and container images using goreleaser.
build-snapshot:
	goreleaser --snapshot --rm-dist --config .github/.goreleaser-develop.yaml --skip-validate

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
