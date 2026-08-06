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
	@files="$$(find . -type f -name '*.go' -not -path './vendor/*' -exec $(GOFMT) -l {} +)"; \
	if [ -n "$$files" ]; then \
		printf '%s\n' "Go files need formatting:" "$$files"; \
		exit 1; \
	fi

fmt-json: ## Format JSON files with jq.
	@set -eu; \
	command -v "$(JQ)" >/dev/null 2>&1 || { \
		printf '%s\n' "JSON formatter not found: $(JQ)" >&2; \
		exit 127; \
	}; \
	find $(JSON_DIRS) -type f -name '*.json' -print | sort | while IFS= read -r file; do \
		tmp="$${file}.tmp.$$$$"; \
		trap 'rm -f "$$tmp"' 0 1 2 3 15; \
		if ! "$(JQ)" --indent "$(JSON_INDENT)" . "$$file" > "$$tmp"; then \
			printf 'Failed to format JSON: %s\n' "$$file" >&2; \
			exit 1; \
		fi; \
		if cmp -s "$$file" "$$tmp"; then \
			rm -f "$$tmp"; \
		else \
			mv "$$tmp" "$$file"; \
		fi; \
		trap - 0 1 2 3 15; \
	done

fmt-json-check: ## Check JSON formatting with jq without changing files.
	@set -eu; \
	command -v "$(JQ)" >/dev/null 2>&1 || { \
		printf '%s\n' "JSON formatter not found: $(JQ)" >&2; \
		exit 127; \
	}; \
	find $(JSON_DIRS) -type f -name '*.json' -print | sort | ( \
		status=0; \
		while IFS= read -r file; do \
			tmp="$$(mktemp)"; \
			if ! "$(JQ)" --indent "$(JSON_INDENT)" . "$$file" > "$$tmp"; then \
				printf 'Invalid JSON: %s\n' "$$file" >&2; \
				status=1; \
			elif ! diff -u "$$file" "$$tmp"; then \
				status=1; \
			fi; \
			rm -f "$$tmp"; \
		done; \
		exit "$$status"; \
	)

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
