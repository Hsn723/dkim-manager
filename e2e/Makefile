PROJECT_NAME = dkim-manager
CERT_MANAGER_VERSION = 1.6.1
KUBERNETES_VERSION = $(shell curl -L -s https://dl.k8s.io/release/stable.txt)
KIND_NODE_TAG = ""

BINDIR = $(dir $(shell pwd))/bin/
HELM = $(BINDIR)/helm
KIND = $(BINDIR)/kind
KUBECTL = $(BINDIR)/kubectl
KIND_CONFIG = kind-config.yaml
KUBECONFIG = $(shell pwd)/.kubeconfig
export KUBECTL KUBECONFIG

.PHONY: start-kind
start-kind:
	$(KIND) create cluster --name=$(PROJECT_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(KIND_NODE_TAG) --wait 1m

.PHONY: deploy
deploy:
	cd ..; CGO_ENABLED=0 go build -o dkim-manager -ldflags="-w -s" cmd/dkim-manager/main.go
	cd ..; docker build --no-cache -t $(PROJECT_NAME):dev .
	$(KIND) load docker-image $(PROJECT_NAME):dev --name=$(PROJECT_NAME)

.PHONY: bootstrap
bootstrap:
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	$(HELM) install --create-namespace --namespace $(PROJECT_NAME) $(PROJECT_NAME) ../charts/$(PROJECT_NAME) -f values.yaml
	$(KUBECTL) -n $(PROJECT_NAME) wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -f ../config/crd/third-party/dnsendpoint.yaml

.PHONY: start
start: $(HELM) $(KIND) $(KUBECTL) crds start-kind deploy bootstrap

.PHONY: test
test:
	env PATH="$$(pwd)/../bin:$$PATH" RUN_E2E=1 \
		go test -v -race . -ginkgo.v -ginkgo.fail-fast

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
	curl -sfL -o $@ https://github.com/kubernetes-sigs/kind/releases/latest/download/kind-linux-amd64
	chmod a+x $@

$(KUBECTL):
	mkdir -p ../bin
	curl -sfL -o $@ https://dl.k8s.io/release/$(KUBERNETES_VERSION)/bin/linux/amd64/kubectl
	chmod a+x $@

.PHONY: get-k8s-versions
get-k8s-versions:
	@LATEST_VERSION=$$(curl -L -s https://dl.k8s.io/release/stable.txt) && \
	KIND_TAGS=$$(gh api /repos/kubernetes-sigs/kind/releases/latest | jq -r '.body' | grep -Po '(?<=- v)(?:\d+\.\d+\.\d+: `kindest\/node:v)\K(\d+\.\d+\.\d+@sha256:[a-f0-9]+)') && \
	LATEST_VERSION_MAJOR=$$(echo $${LATEST_VERSION} | grep -Po '(?<=v)\d+\.\d+') && \
	LATEST_KIND_TAG=$$(printf "%s\n" $${KIND_TAGS} | grep "$${LATEST_VERSION_MAJOR}" | sort -u | tail -n 1) && \
	PREVIOUS_VERSION_MAJOR=$$(echo $${LATEST_VERSION_MAJOR} | awk -F. -v OFS=. '{$$NF -= 1; print}') && \
	PREVIOUS_KIND_TAG=$$(printf "%s\n" $${KIND_TAGS} | grep "$${PREVIOUS_VERSION_MAJOR}" | sort -u | tail -n 1) && \
	PREVIOUS2_VERSION_MAJOR=$$(echo $${PREVIOUS_VERSION_MAJOR} | awk -F. -v OFS=. '{$$NF -= 1; print}') && \
	PREVIOUS2_KIND_TAG=$$(printf "%s\n" $${KIND_TAGS} | grep "$${PREVIOUS2_VERSION_MAJOR}" | sort -u | tail -n 1) && \
	echo "['$${PREVIOUS2_KIND_TAG}', '$${PREVIOUS_KIND_TAG}', '$${LATEST_KIND_TAG}']"
