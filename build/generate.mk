#
# Makefile Fragment for generating code
#

CONTROLLER_GEN    ?= controller-gen
HEADER_FILE       ?= $(SRCDIR)/hack/boilerplate.go.txt

GOTOOLS += sigs.k8s.io/controller-tools/cmd/controller-gen


# Generate code
generate: tools
	@echo "=== $(PROJECT_NAME) === [ generate         ]: Running $(CONTROLLER_GEN)..."
	@$(CONTROLLER_GEN) object:headerFile=$(HEADER_FILE) paths="./..."

