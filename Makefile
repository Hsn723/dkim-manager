
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/hsn723/dkim-manager:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
CST_VERSION = 1.10.0
ENVTEST_K8S_VERSION = 1.23
EXTERNAL_DNS_VERSION = 0.10.2
HELM_VERSION = 3.7.1
KUSTOMIZE_VERSION = 4.4.1

BINDIR = $(shell pwd)/bin

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: kustomize controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build config/helm/overlays/crds > charts/dkim-manager/crds/dkim-manager.atelierhsn.com_dkimkeys.yaml
	$(KUSTOMIZE) build config/helm/overlays/templates > charts/dkim-manager/templates/generated/generated.yaml

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint:
	if [ -z "$(shell which pre-commit)" ]; then pip3 install pre-commit; fi
	pre-commit install
	pre-commit run --all-files

crds:
	mkdir -p config/crd/third-party
	curl -fsL -o config/crd/third-party/dnsendpoint.yaml -sLf https://raw.githubusercontent.com/kubernetes-sigs/external-dns/v$(EXTERNAL_DNS_VERSION)/docs/contributing/crd-source/crd-manifest.yaml

.PHONY: test
test: manifests generate fmt vet crds setup-envtest ## Run tests.
	go test -v -count 1 -race ./pkg/... -coverprofile pkg-cover.out
	source <($(SETUP_ENVTEST) use -p env); \
		go test -v -count 1 -race ./controllers -ginkgo.progress -ginkgo.v -ginkgo.failFast -coverprofile controllers-cover.out
	source <($(SETUP_ENVTEST) use -p env); \
		go test -v -count 1 -race ./hooks -ginkgo.progress -ginkgo.v -ginkgo.failFast -coverprofile hooks-cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	CGO_ENABLED=0 go build -o $(BINDIR)/dkim-manager -ldflags="-w -s" cmd/dkim-manager/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

CONTROLLER_GEN = $(BINDIR)/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)

HELM :=  $(BINDIR)/helm
.PHONY: helm
helm: $(HELM) ## Download helm locally if necessary.

$(HELM):
	mkdir -p $(BINDIR)
	curl -L -sS https://get.helm.sh/helm-v$(HELM_VERSION)-linux-amd64.tar.gz \
	  | tar xz -C $(BINDIR) --strip-components 1 linux-amd64/helm


KUSTOMIZE = $(BINDIR)/kustomize
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.

$(KUSTOMIZE):
	mkdir -p $(BINDIR)
	curl -fsL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | \
	tar -C $(BINDIR) -xzf -

SETUP_ENVTEST = $(BINDIR)/setup-envtest
.PHONY: setup-envtest
setup-envtest: $(SETUP_ENVTEST)
$(SETUP_ENVTEST): ## Download envtest-setup locally if necessary.
	GOBIN=$(BINDIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

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

.PHONY: setup-container-structure-test
setup-container-structure-test:
	if [ -z "$(shell which container-structure-test)" ]; then \
		curl -LO https://storage.googleapis.com/container-structure-test/v$(CST_VERSION)/container-structure-test-linux-amd64 && mv container-structure-test-linux-amd64 container-structure-test && chmod +x container-structure-test && sudo mv container-structure-test /usr/local/bin/; \
	fi

.PHONY: container-structure-test
container-structure-test: setup-container-structure-test
	printf "amd64\narm64" | xargs -n1 -I {} container-structure-test test --image ghcr.io/hsn723/dkim-manager:$(shell git describe --tags --abbrev=0 || echo v0.0.0)-next-{} --config cst.yaml
