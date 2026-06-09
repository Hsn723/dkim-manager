include Makefile.common

# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/hsn723/dkim-manager:latest

BINDIR = $(shell pwd)/bin

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
manifests: init-aqua ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	kustomize build config/helm/overlays/crds > charts/dkim-manager/templates/generated/crds/dkim-manager.atelierhsn.com_dkimkeys.yaml
	kustomize build config/helm/overlays/templates > charts/dkim-manager/templates/generated/generated.yaml

.PHONY: update-external-dns
update-external-dns: init-aqua ## Update the external-dns Helm chart dependency to the latest version.
	helm dependency update charts/dkim-manager

.PHONY: update-chart
update-chart: update-external-dns manifests ## Update the Helm chart version and appVersion in Chart.yaml.
	APP_VERSION=$$(cat VERSION) yq e -i '.appVersion = strenv(APP_VERSION)' charts/dkim-manager/Chart.yaml
	if [ "$$(git diff --name-only charts/dkim-manager/Chart.yaml)" != "" ]; then \
		yq e -i '.version |= (split(".") | .[-1] |= ((. tag = "!!int") + 1) | join("."))' charts/dkim-manager/Chart.yaml ; \
	fi

.PHONY: generate
generate: init-aqua ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: init-aqua ## Run linters.
	pre-commit install
	pre-commit run --all-files

crds: init-aqua ## Generate CRD manifests for third-party dependencies.
	mkdir -p config/crd/third-party
	@if ! helm repo list | grep -q "external-dns"; then \
		helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/ ;\
	fi
	helm repo update
	helm show crds external-dns/external-dns --version "$$(yq .dependencies[0].version charts/dkim-manager/Chart.yaml)" > config/crd/third-party/dnsendpoint.yaml

.PHONY: test
test: init-aqua manifests generate fmt vet crds ## Run tests.
	go test -v -count 1 -race ./pkg/... -coverprofile pkg-cover.out
	source <(setup-envtest use -p env); \
		go test -v -count 1 -race ./controllers -ginkgo.v -ginkgo.fail-fast -coverprofile controllers-cover.out
	source <(setup-envtest use -p env); \
		go test -v -count 1 -race ./hooks -ginkgo.v -ginkgo.fail-fast -coverprofile hooks-cover.out

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
install: init-aqua manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kustomize build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: init-aqua manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kustomize build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: init-aqua manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kustomize build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

$(BINDIR):
	mkdir -p $(BINDIR)

.PHONY: container-structure-test
container-structure-test: init-aqua
	yq '.builds[0] | .goarch[]' .goreleaser.yml | xargs -I {} container-structure-test test --image ghcr.io/hsn723/dkim-manager:$(shell git describe --tags --abbrev=0 --match "v*" || echo v0.0.0)-next-{} --platform linux/{} --config cst.yaml
