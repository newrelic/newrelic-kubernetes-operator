#
# Makefile fragment for Linting
#

GO           ?= go
MISSPELL     ?= misspell
GOFMT        ?= gofmt
GOIMPORTS    ?= goimports

COMMIT_LINT_CMD   ?= go-gitlint
COMMIT_LINT_REGEX ?= "(bug|chore|docs|feat|fix|refactor|tests?)(\([^\)]+\))?: .*"
COMMIT_LINT_START ?= "2020-06-05"

GOLINTER      = golangci-lint

EXCLUDEDIR      ?= .git
SRCDIR          ?= .
GO_PKGS         ?= $(shell ${GO} list ./... | grep -v -e "/vendor/" -e "/example")
FILES           ?= $(shell find ${SRCDIR} -type f | grep -v -e '.git/' -e '/vendor/')
GO_FILES        ?= $(shell find $(SRCDIR) -type f -name "*.go" | grep -v -e ".git/" -e '/vendor/' -e '/example/')
PROJECT_MODULE  ?= $(shell $(GO) list -m)

GO_MOD_OUTDATED ?= go-mod-outdated

GOTOOLS += github.com/client9/misspell/cmd/misspell \
           github.com/llorllale/go-gitlint \
           github.com/psampaz/go-mod-outdated \
           golang.org/x/tools/cmd/goimports

# golangci-lint version and installation
GOLANGCI_LINT_VERSION ?= v1.62.2

lint: outdated spell-check gofmt golangci lint-commit goimports
lint-fix: spell-check-fix gofmt-fix goimports

#
# Check spelling on all the files, not just source code
#
spell-check: tools
	@echo "=== $(PROJECT_NAME) === [ spell-check      ]: Checking for spelling mistakes with $(MISSPELL)..."
	@$(MISSPELL) -source text $(FILES)

spell-check-fix: tools
	@echo "=== $(PROJECT_NAME) === [ spell-check-fix  ]: Fixing spelling mistakes with $(MISSPELL)..."
	@$(MISSPELL) -source text -w $(FILES)

gofmt: tools
	@echo "=== $(PROJECT_NAME) === [ gofmt            ]: Checking file format with $(GOFMT)..."
	@find . -path "$(EXCLUDEDIR)" -prune -print0 | xargs -0 $(GOFMT) -e -l -s -d ${SRCDIR}

gofmt-fix: tools
	@echo "=== $(PROJECT_NAME) === [ gofmt-fix        ]: Fixing file format with $(GOFMT)..."
	@find . -path "$(EXCLUDEDIR)" -prune -print0 | xargs -0 $(GOFMT) -e -l -s -w ${SRCDIR}

goimports: tools
	@echo "=== $(PROJECT_NAME) === [ goimports        ]: Checking imports with $(GOIMPORTS)..."
	@$(GOIMPORTS) -w -local $(PROJECT_MODULE) $(GO_FILES)

lint-commit: tools
	@echo "=== $(PROJECT_NAME) === [ lint-commit      ]: Checking that commit messages are properly formatted ($(COMMIT_LINT_CMD))..."
	@$(COMMIT_LINT_CMD) --since=$(COMMIT_LINT_START) --subject-minlen=10 --subject-maxlen=120 --subject-regex=$(COMMIT_LINT_REGEX)

golangci: tools install-golangci-lint
	@echo "=== $(PROJECT_NAME) === [ golangci-lint    ]: Linting using $(GOLINTER)..."
	@$(GOLINTER) run

install-golangci-lint:
	@if ! command -v $(GOLINTER) >/dev/null 2>&1 || ! $(GOLINTER) version 2>/dev/null | grep -q $(GOLANGCI_LINT_VERSION); then \
		echo "=== $(PROJECT_NAME) === [ golangci-lint    ]: Installing $(GOLINTER) $(GOLANGCI_LINT_VERSION) into $$($(GO) env GOPATH)/bin..."; \
		GOBIN=$$($(GO) env GOPATH)/bin $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) || exit 1; \
		echo "Installed $(GOLINTER) $(GOLANGCI_LINT_VERSION) built with $$($(GO) version | awk '{print $$3}')"; \
	fi

outdated: tools
	@echo "=== $(PROJECT_NAME) === [ outdated         ]: Finding outdated deps with $(GO_MOD_OUTDATED)..."
	@$(GO) list -u -m -json all | $(GO_MOD_OUTDATED) -direct -update

.PHONY: lint spell-check spell-check-fix gofmt gofmt-fix lint-fix lint-commit outdated goimports install-golangci-lint
