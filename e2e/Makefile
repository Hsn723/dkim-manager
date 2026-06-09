include ../Makefile.common

PROJECT_NAME = dkim-manager
CERT_MANAGER_VERSION = 1.20.2
KUBERNETES_VERSION = $(shell curl --retry 3 --retry-all-errors -L -s https://dl.k8s.io/release/stable.txt)
KIND_NODE_TAG = ""

BINDIR = $(dir $(shell pwd))/bin/
KUBECTL = $(BINDIR)/kubectl
KIND_CONFIG = kind-config.yaml
KUBECONFIG = $(shell pwd)/.kubeconfig
export KUBECTL KUBECONFIG

.PHONY: start-kind
start-kind:
	kind create cluster --name=$(PROJECT_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(KIND_NODE_TAG) --wait 1m

.PHONY: deploy
deploy:
	cd ..; CGO_ENABLED=0 go build -o dkim-manager -ldflags="-w -s" cmd/dkim-manager/main.go
	cd ..; docker build --no-cache -t $(PROJECT_NAME):dev .
	kind load docker-image $(PROJECT_NAME):dev --name=$(PROJECT_NAME)

.PHONY: bootstrap
bootstrap: init-aqua
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	helm install --create-namespace --namespace $(PROJECT_NAME) $(PROJECT_NAME) ../charts/$(PROJECT_NAME) -f values.yaml
	$(KUBECTL) -n $(PROJECT_NAME) wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -f ../config/crd/third-party/dnsendpoint.yaml

.PHONY: start
start: init-aqua $(KUBECTL) crds start-kind deploy bootstrap

.PHONY: test
test:
	env PATH="$$(pwd)/../bin:$$PATH" RUN_E2E=1 \
		go test -v -race . -ginkgo.v -ginkgo.fail-fast

.PHONY: stop
stop:
	kind delete cluster --name=$(PROJECT_NAME)
	-docker image rm $(PROJECT_NAME):dev
	-docker image prune -f

.PHONY: crds
crds:
	$(MAKE) -C .. crds

$(HELM):
	$(MAKE) -C .. helm

$(KUBECTL):
	mkdir -p ../bin
	curl --retry 3 --retry-all-errors -sfL -o $@ https://dl.k8s.io/release/$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $@

.PHONY: get-k8s-versions
get-k8s-versions:
	@KIND_TAGS=$$(gh api /repos/kubernetes-sigs/kind/releases/latest --jq '.body' | grep -Po '(?:v\d+\.\d+\.\d+: `kindest\/node:v)\K(\d+\.\d+\.\d+@sha256:[a-f0-9]+)' | sort -r | head -3) && \
	printf "[%s]" "$$(printf "'%s', " $${KIND_TAGS} | sed 's/, $$//')"
