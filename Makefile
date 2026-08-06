.DEFAULT_GOAL := check

GO ?= go
GOFMT ?= gofmt
JQ ?= jq
PACKAGES ?= ./...
JSON_DIRS ?= schemas testdata
JSON_INDENT ?= 2
COVERAGE_FILE ?= coverage.out

.PHONY: check build test test-race vet fmt fmt-check fmt-json fmt-json-check tidy mod-check coverage clean help

check: fmt-check vet test ## Run formatting checks, vet, and tests.

build: ## Build all Go packages.
	$(GO) build $(PACKAGES)

test: ## Run all tests.
	$(GO) test $(PACKAGES)

test-race: ## Run all tests with the race detector.
	$(GO) test -race $(PACKAGES)

vet: ## Run go vet on all packages.
	$(GO) vet $(PACKAGES)

fmt: fmt-json ## Format all Go and JSON files.
	$(GO) fmt $(PACKAGES)

fmt-check: fmt-json-check ## Check Go and JSON formatting without changing files.
	@GOFMT="$(GOFMT)" ./scripts/fmt-go-check.sh

fmt-json: ## Format JSON files with jq.
	@JQ="$(JQ)" JSON_INDENT="$(JSON_INDENT)" ./scripts/fmt-json.sh $(JSON_DIRS)

fmt-json-check: ## Check JSON formatting with jq without changing files.
	@JQ="$(JQ)" JSON_INDENT="$(JSON_INDENT)" ./scripts/fmt-json-check.sh $(JSON_DIRS)

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
