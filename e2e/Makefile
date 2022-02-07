PROJECT_NAME = dkim-manager
CERT_MANAGER_VERSION = 1.6.1
KIND_VERSION = 0.11.1
KUBERNETES_VERSION = 1.23.1

BINDIR = $(dir $(shell pwd))/bin/
HELM = $(BINDIR)/helm
KIND = $(BINDIR)/kind
KUBECTL = $(BINDIR)/kubectl
KIND_CONFIG = kind-config.yaml
KUBECONFIG = $(shell pwd)/.kubeconfig
export KUBECTL KUBECONFIG

.PHONY: start
start: $(HELM) $(KIND) $(KUBECTL) crds
	$(KIND) create cluster --name=$(PROJECT_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(KUBERNETES_VERSION) --wait 1m
	cd ..; CGO_ENABLED=0 go build -o dkim-manager -ldflags="-w -s" cmd/dkim-manager/main.go
	cd ..; docker build --no-cache -t $(PROJECT_NAME):dev .
	$(KIND) load docker-image $(PROJECT_NAME):dev --name=$(PROJECT_NAME)
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	$(HELM) install --create-namespace --namespace $(PROJECT_NAME) $(PROJECT_NAME) ../charts/$(PROJECT_NAME) -f values.yaml
	$(KUBECTL) -n $(PROJECT_NAME) wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -f ../config/crd/third-party/dnsendpoint.yaml

.PHONY: test
test:
	env PATH=$$(pwd)/../bin:$$PATH RUN_E2E=1 \
		go test -v -race . -ginkgo.progress -ginkgo.v -ginkgo.fail-fast

.PHONY: stop
stop: $(KIND) 
	$(KIND) delete cluster --name=$(PROJECT_NAME)
	-docker image rm $(PROJECT_NAME):dev
	-docker image prune -f

.PHONY: crds
crds:
	$(MAKE) -C .. crds

$(HELM):
	$(MAKE) -C .. helm

$(KIND):
	mkdir -p ../bin
	curl -sfL -o $@ https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-linux-amd64
	chmod a+x $@

$(KUBECTL):
	mkdir -p ../bin
	curl -sfL -o $@ https://dl.k8s.io/release/v$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $@
