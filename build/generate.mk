#
# Makefile Fragment for generating code
#

CONTROLLER_GEN    ?= controller-gen
HEADER_FILE       ?= $(SRCDIR)/build/generate/boilerplate.go.txt

GOTOOLS += sigs.k8s.io/controller-tools/cmd/controller-gen


# Generate code
generate: tools
	@echo "=== $(PROJECT_NAME) === [ generate         ]: Running $(CONTROLLER_GEN)..."
	@$(CONTROLLER_GEN) object:headerFile=$(HEADER_FILE) paths="./..."
	@cd interfaces/ && $(GO) generate

