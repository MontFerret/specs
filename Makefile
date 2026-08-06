.DEFAULT_GOAL := check

GO ?= go
GOFMT ?= gofmt
PACKAGES ?= ./...
COVERAGE_FILE ?= coverage.out

.PHONY: check build test test-race vet fmt fmt-check tidy mod-check coverage clean help

check: fmt-check vet test ## Run formatting checks, vet, and tests.

build: ## Build all Go packages.
	$(GO) build $(PACKAGES)

test: ## Run all tests.
	$(GO) test $(PACKAGES)

test-race: ## Run all tests with the race detector.
	$(GO) test -race $(PACKAGES)

vet: ## Run go vet on all packages.
	$(GO) vet $(PACKAGES)

fmt: ## Format all Go packages.
	$(GO) fmt $(PACKAGES)

fmt-check: ## Check Go formatting without changing files.
	@files="$$(find . -type f -name '*.go' -not -path './vendor/*' -exec $(GOFMT) -l {} +)"; \
	if [ -n "$$files" ]; then \
		printf '%s\n' "Go files need formatting:" "$$files"; \
		exit 1; \
	fi

tidy: ## Update go.mod and go.sum.
	$(GO) mod tidy

mod-check: ## Check module metadata without changing files.
	$(GO) mod tidy -diff

coverage: ## Run tests and write a coverage profile.
	$(GO) test -coverprofile=$(COVERAGE_FILE) $(PACKAGES)

clean: ## Clear the Go test cache and remove the coverage profile.
	$(GO) clean -testcache
	$(RM) $(COVERAGE_FILE)

help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
