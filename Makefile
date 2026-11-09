# Entry point for the whole repository. The recipes live next to the code
# (chameleon/Makefile for Go, cargo for the Rust core); this file only delegates.

.DEFAULT_GOAL := help
.PHONY: help build test test-core test-go test-integration test-all clean tag

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  make %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo
	@echo "Release: make tag [VERSION=X.Y.Z]   (see DEVELOPERS.md)"

build: ## Build the Rust core and the CLI (chameleon/bin/chameleon)
	$(MAKE) -C chameleon build

test: test-core test-go ## Unit tests: Rust core and Go runtime

test-core: ## Rust core tests (cargo test)
	cd chameleon-core && cargo test

test-go: ## Go unit tests (builds the Rust library first)
	$(MAKE) -C chameleon test

test-integration: ## Go integration tests against PostgreSQL 16 (needs Docker with Compose)
	$(MAKE) -C chameleon test-integration

test-all: test-core ## Rust tests plus all Go tests, integration included
	$(MAKE) -C chameleon test-all

clean: ## Remove build outputs
	$(MAKE) -C chameleon clean

tag: ## Bump the version, tag and push a release. Optional: VERSION=X.Y.Z
	./scripts/tag-release.sh $(VERSION)
