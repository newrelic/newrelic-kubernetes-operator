#
# Makefile snippet for install/uninstall of the operator
#

# Image URL to use all building/pushing image targets
DOCKER_IMAGE ?= newrelic/kubernetes-operator:snapshot
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"


# Install CRDs into a cluster
install: manifests
	@echo "=== $(PROJECT_NAME) === [ install          ]: Applying operator..."
	@kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	@echo "=== $(PROJECT_NAME) === [ uninstall        ]: Deleting operator..."
	@kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests docker-build
	@echo "=== $(PROJECT_NAME) === [ deploy           ]: Deploying operator as docker image ${DOCKER_IMAGE}..."
	@cd config/manager && kustomize edit set image controller=${DOCKER_IMAGE}
	@kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: tools
	@echo "=== $(PROJECT_NAME) === [ manifests        ]: Generating manifests..."
	@$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

