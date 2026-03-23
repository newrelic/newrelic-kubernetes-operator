RELEASE_SCRIPT ?= ./scripts/release.sh

# goreleaser is installed on-demand due to dependency conflicts
GORELEASER_VERSION ?= v0.143.0

REL_CMD ?= goreleaser
DIST_DIR ?= ./dist

# Example usage: make release version=0.11.0
release: build
	@echo "=== $(PROJECT_NAME) === [ release          ]: Generating release."
	$(RELEASE_SCRIPT) $(version)

release-clean:
	@echo "=== $(PROJECT_NAME) === [ release-clean    ]: distribution files..."
	@rm -rfv $(DIST_DIR) $(SRCDIR)/tmp

release-publish: clean tools docker-login release-notes
	@echo "=== $(PROJECT_NAME) === [ release-publish  ]: Installing $(REL_CMD) $(GORELEASER_VERSION)..."
	@$(GO) install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)
	@echo "=== $(PROJECT_NAME) === [ release-publish  ]: Publishing release via $(REL_CMD)"
	$(REL_CMD) --release-notes=$(SRCDIR)/tmp/$(RELEASE_NOTES_FILE)

# Local Snapshot
snapshot: release-clean
	@echo "=== $(PROJECT_NAME) === [ snapshot         ]: Installing $(REL_CMD) $(GORELEASER_VERSION)..."
	@$(GO) install github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)
	@echo "=== $(PROJECT_NAME) === [ snapshot         ]: Creating release via $(REL_CMD)"
	@echo "=== $(PROJECT_NAME) === [ snapshot         ]:   THIS WILL NOT BE PUBLISHED!"
	$(REL_CMD) --skip-publish --snapshot


.PHONY: release release-clean release-homebrew release-publish snapshot
