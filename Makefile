
#all: manager


# Build manager binary
#manager: generate fmt vet
#	go build -ldflags "-X main.NewRelicAPIKey=${NEWRELIC_API_KEY}" -o bin/manager main.go



# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
#CRD_OPTIONS ?= "crd:trivialVersions=true"

# Install CRDs into a cluster
#install: manifests
#	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
#uninstall: manifests
#	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
#deploy: manifests
#	cd config/manager && kustomize edit set image controller=${IMG}
#	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
#manifests: controller-gen
#	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases




#############################
# Global vars
#############################
PROJECT_NAME := $(shell basename $(shell pwd))
PROJECT_VER  ?= $(shell git describe --tags --always --dirty | sed -e '/^v/s/^v\(.*\)$$/\1/g') # Strip leading 'v' if found
# Last released version (not dirty)
PROJECT_VER_TAGGED  ?= $(shell git describe --tags --always --abbrev=0 | sed -e '/^v/s/^v\(.*\)$$/\1/g') # Strip leading 'v' if found

SRCDIR       ?= .
GO            = go

# The root module (from go.mod)
PROJECT_MODULE  ?= $(shell $(GO) list -m)

#############################
# Targets
#############################
all: build

# Humans running make:
build: check-version clean generate lint test cover-report compile

# Build command for CI tooling
build-ci: check-version clean lint test compile-only

# All clean commands
clean: cover-clean compile-clean release-clean

# Import fragments
include build/compile.mk
include build/docker.mk
include build/document.mk
include build/generate.mk
include build/lint.mk
include build/release.mk
include build/test.mk
include build/tools.mk
include build/util.mk

.PHONY: all build build-ci clean
