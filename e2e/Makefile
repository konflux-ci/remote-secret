MAKEFLAGS   += --warn-undefined-variables
SHELL       := /bin/bash
.SHELLFLAGS := -euo pipefail -c
GOARCH ?= $(shell go env GOARCH)
KIND_IMG       = "kindest/node:v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb"
DOCKER_BUILD_ARGS     ?=

export E2E_IMAGE_NAME ?= quay.io/redhat-appstudio/remote-secret-controller-e2e
export GINKGO_LABELS ?= !managed
export TEST_SUITES ?= provider

start-kind: ## Start kind cluster
	kind create cluster \
	  --name remote-secret \
	  --config kind.yaml \
	  --retain \
	  --image "$(KIND_IMG)"

test: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		IMAGE_NAME=$(IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=$(GOARCH) \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=$(GOARCH) --build-arg TARGETOS=linux"
	kind load docker-image --name="remote-secret" $(IMG):$(VERSION)
	kind load docker-image --name="remote-secret" $(E2E_IMAGE_NAME):$(VERSION)
	./run.sh

test.managed: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		VERSION=$(VERSION) \
		ARCH=$(GOARCH) \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=$(GOARCH) --build-arg TARGETOS=linux"
	$(MAKE) -C ../ docker.push \
		VERSION=$(VERSION)
	$(MAKE) -C ../ docker.push \
		IMAGE_NAME=$(E2E_IMAGE_NAME) \
		VERSION=$(VERSION)
	./run.sh

e2e-bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH)  ginkgo build ./suites/...

e2e-image: e2e-bin
	-rm -rf ./k8s/deploy
	mkdir -p k8s
	$(MAKE) -C ../ helm.generate
	cp -r ../deploy ./k8s
	docker build $(DOCKER_BUILD_ARGS) --build-arg=TARGETARCH=$(GOARCH)  -t $(E2E_IMAGE_NAME):$(VERSION)   .

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name remote-secret \

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
