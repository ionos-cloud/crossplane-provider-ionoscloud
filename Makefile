# ====================================================================================
# Setup Project
# If you are building the provider locally for the first time and you see macro errors related to the makefile, run `make submodules`
PROJECT_NAME := crossplane-provider-ionoscloud
PROJECT_REPO := github.com/ionos-cloud/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# Setup Output
-include build/makelib/output.mk

# Setup Golang
NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
-include build/makelib/golang.mk

# Setup Kubernetes tools
-include build/makelib/k8s_tools.mk

# Setup Images
DOCKER_REGISTRY ?= crossplane
REGISTRY ?= ghcr.io
ORG_NAME ?= ionos-cloud
VERSION ?= latest
PROVIDER_IMAGE=$(REGISTRY)/$(ORG_NAME)/$(PROJECT_NAME)
PACKAGE_PROVIDER_IMAGE=$(PROVIDER_IMAGE):$(VERSION)
CONTROLLER_IMAGE=$(REGISTRY)/$(ORG_NAME)/$(PROJECT_NAME)-controller
PACKAGE_CONTROLLER_IMAGE=$(CONTROLLER_IMAGE):$(VERSION)
IMAGE_PATH=$(REGISTRY)/$(ORG_NAME)/$(PROVIDER_NAME)
PKG_PATH=$(REGISTRY)/$(ORG_NAME)/$(PKG_NAME)
IMAGES = $(PROJECT_NAME) $(PROJECT_NAME)-controller
-include build/makelib/image.mk

# Setup documentation
DOCS_OUT?=$(shell pwd)/docs/api

.PHONY: docs.update
docs.update:
	@DOCS_OUT=${DOCS_OUT} go run tools/doc/main.go

.PHONY: docker.list
docker.list:
	@docker image list

.PHONY: docker.tag
docker.tag:
	@docker tag $(PROVIDER_IMAGE) $(PROVIDER_IMAGE):$(VERSION)
	@docker tag $(CONTROLLER_IMAGE) $(CONTROLLER_IMAGE):$(VERSION)

.PHONY: docker.push
docker.push:
	@docker push $(PROVIDER_IMAGE):$(VERSION)
	@docker push $(CONTROLLER_IMAGE):$(VERSION)

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

crds.clean:
	@$(INFO) cleaning generated CRDs
	@find package/crds -name *.yaml -exec sed -i.sed -e '1,2d' {} \; || $(FAIL)
	@find package/crds -name *.yaml.sed -delete || $(FAIL)
	@$(OK) cleaned generated CRDs

generate: crds.clean

# integration tests
e2e.run: test-integration

# Run integration tests.
test-integration: $(KIND) $(KUBECTL) $(UP) $(HELM3)
	@$(INFO) running integration tests using kind $(KIND_VERSION)
	@$(ROOT_DIR)/cluster/local/integration_tests.sh VERSION=$(VERSION) || $(FAIL)
	@$(OK) integration tests passed

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

dev: $(KIND) $(KUBECTL)
	@$(INFO) Creating kind cluster
	@$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	@$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Crossplane CRDs
	@$(KUBECTL) create -k https://github.com/crossplane/crossplane//cluster?ref=master
	@$(INFO) Installing Provider IONOS Cloud CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider IONOS Cloud controllers
	@$(GO) run cmd/provider/main.go --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallthrough test-integration run crds.clean dev dev-clean

# ====================================================================================
# Special Targets

# Install gomplate
GOMPLATE_VERSION := 3.10.0
GOMPLATE := $(TOOLS_HOST_DIR)/gomplate-$(GOMPLATE_VERSION)

$(GOMPLATE):
	@$(INFO) installing gomplate $(SAFEHOSTPLATFORM)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(GOMPLATE) https://github.com/hairyhenderson/gomplate/releases/download/v$(GOMPLATE_VERSION)/gomplate_$(SAFEHOSTPLATFORM) || $(FAIL)
	@chmod +x $(GOMPLATE)
	@$(OK) installing gomplate $(SAFEHOSTPLATFORM)

export GOMPLATE

# This target prepares repo for your provider by replacing all "template"
# occurrences with your provider name.
# This target can only be run once, if you want to rerun for some reason,
# consider stashing/resetting your git state.
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
provider.prepare:
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@PROVIDER=$(provider) ./hack/helpers/prepare.sh

# This target adds a new api type and its controller.
# You would still need to register new api in "apis/<provider>.go" and
# controller in "internal/controller/<provider>.go".
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
#   group: API group for the type you want to add.
#   kind: Kind of the type you want to add
#	apiversion: API version of the type you want to add. Optional and defaults to "v1alpha1"
provider.addtype: $(GOMPLATE)
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@[ "${group}" ] || ( echo "argument \"group\" is not set"; exit 1 )
	@[ "${kind}" ] || ( echo "argument \"kind\" is not set"; exit 1 )
	@PROVIDER=$(provider) GROUP=$(group) KIND=$(kind) APIVERSION=$(apiversion) PROJECT_REPO=$(PROJECT_REPO) ./hack/helpers/addtype.sh

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special
