#
# Makefile fragment for installing deps
#

GO           ?= go

# These should be mirrored in /tools.go to keep versions consistent
GOTOOLS      += github.com/client9/misspell/cmd/misspell

tools: check-version
	@echo "=== $(PROJECT_NAME) === [ tools            ]: Installing tools required by the project..."
	@$(GO) install $(GOTOOLS)

tools-update: check-version
	@echo "=== $(PROJECT_NAME) === [ tools-update     ]: Updating tools required by the project..."
	@$(GO) get -u $(GOTOOLS)


.PHONY: tools tools-update
