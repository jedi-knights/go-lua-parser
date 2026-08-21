# Makefile for go-lua-parser
#
# Local dev commands mirror .github/workflows/ci.yml so `make lint`,
# `make test-ci`, and `make coverage-check` verify the same things CI does.
# The coverage targets mirror .github/workflows/badge.yaml (LCOV via
# gcov2lcov + lcov/genhtml).

# Force bash so recipes can use `set -o pipefail` and `[[ ... ]]`. Make's
# default is /bin/sh, which on Ubuntu is dash and rejects these features —
# CI would fail with "Illegal option -o pipefail" otherwise.
SHELL := /bin/bash

GO             ?= go
GOLANGCI_LINT  ?= golangci-lint
GCOV2LCOV      ?= gcov2lcov
LCOV           ?= lcov
GENHTML        ?= genhtml

COVERAGE_OUT          := coverage.out
LCOV_DIR              := coverage
LCOV_FILE             := $(LCOV_DIR)/lcov.info
HTML_DIR              := htmlcov
TEST_RESULTS_JSON     := test-results.json

# Minimum acceptable line coverage. CI passes this via
# `make coverage-check COVERAGE_THRESHOLD=<n>` to gate merges.
COVERAGE_THRESHOLD    ?= 84

.PHONY: help lint test test-ci test-coverage test-html coverage-check \
        vet verify build clean install-ci-deps \
        require-golangci-lint require-gcov2lcov require-lcov require-genhtml

# Default target
help:
	@echo "go-lua-parser development commands"
	@echo ""
	@echo "Quality:"
	@echo "  make lint            - Run golangci-lint (matches CI)"
	@echo "  make vet             - Run go vet"
	@echo "  make verify          - Verify go.mod checksums"
	@echo ""
	@echo "Testing:"
	@echo "  make test            - Run unit tests with -race (matches CI)"
	@echo "  make test-ci         - Run tests with -json output for CI publishers"
	@echo "  make test-coverage   - Report LCOV line coverage (matches CI gate)"
	@echo "  make test-html       - Generate HTML coverage report under $(HTML_DIR)/"
	@echo "  make coverage-check  - Fail if coverage < COVERAGE_THRESHOLD (default $(COVERAGE_THRESHOLD))"
	@echo ""
	@echo "Build:"
	@echo "  make build           - Build all packages"
	@echo ""
	@echo "Housekeeping:"
	@echo "  make install-ci-deps - Install lcov + gcov2lcov (CI runner use)"
	@echo "  make clean           - Remove coverage artifacts"
	@echo "  make help            - Show this help"

lint: require-golangci-lint
	@$(GOLANGCI_LINT) run ./...

vet:
	@$(GO) vet ./...

verify:
	@$(GO) mod verify

build:
	@$(GO) build ./...

# Human-readable test run. Produces coverage.out.
test: $(COVERAGE_OUT)

$(COVERAGE_OUT):
	@$(GO) test -race -count=1 -coverprofile=$(COVERAGE_OUT) ./...

# CI variant: -json stream tee'd to test-results.json for
# jedi-knights/publish-test-results, while still producing coverage.out.
# Uses the same underlying `go test` invocation as `make test`.
test-ci:
	@set -o pipefail; $(GO) test -race -count=1 -coverprofile=$(COVERAGE_OUT) -json ./... | tee $(TEST_RESULTS_JSON)

# Convert coverage.out to LCOV. File-based dependency: if coverage.out
# is missing, Make runs the recipe above; if lcov.info already exists
# and coverage.out is older, it is not regenerated.
$(LCOV_FILE): $(COVERAGE_OUT) | require-gcov2lcov
	@mkdir -p $(LCOV_DIR)
	@$(GCOV2LCOV) -infile=$(COVERAGE_OUT) -outfile=$(LCOV_FILE)

# Compute and print the LCOV line coverage — same metric CI gates on.
test-coverage: $(LCOV_FILE) | require-lcov
	@COVERAGE=$$($(LCOV) --summary $(LCOV_FILE) 2>&1 | grep -i '^\s*lines' | grep -oE '[0-9]+\.[0-9]+' | head -1); \
		echo "Line coverage: $${COVERAGE}%"

# Gate: fail if coverage < COVERAGE_THRESHOLD. CI invokes this after
# test-ci to enforce the merge floor; developers can run it locally to
# preview whether their branch will pass.
coverage-check: $(LCOV_FILE) | require-lcov
	@COVERAGE=$$($(LCOV) --summary $(LCOV_FILE) 2>&1 | grep -i '^\s*lines' | grep -oE '[0-9]+\.[0-9]+' | head -1); \
		echo "Line coverage: $${COVERAGE}%"; \
		if awk "BEGIN { exit !($${COVERAGE} < $(COVERAGE_THRESHOLD)) }"; then \
			echo "::error::Coverage $${COVERAGE}% is below the $(COVERAGE_THRESHOLD)% threshold"; \
			exit 1; \
		fi

# Local HTML report — same generator (genhtml) that produces the deployed
# report at https://jedi-knights.github.io/go-lua-parser/.
test-html: $(LCOV_FILE) | require-genhtml
	@$(GENHTML) $(LCOV_FILE) -o $(HTML_DIR)/ --quiet --ignore-errors source
	@echo "Report: $(HTML_DIR)/index.html"

# Install the CI-only external tools (lcov + gcov2lcov). Meant for the
# GitHub Actions runner; on a dev machine, install once via brew/apt.
install-ci-deps:
	@sudo apt-get install -y --quiet lcov
	@$(GO) install github.com/jandelgado/gcov2lcov@latest

clean:
	@rm -rf $(COVERAGE_OUT) $(LCOV_DIR) $(HTML_DIR) $(TEST_RESULTS_JSON)

# Tool prerequisites — fail with an actionable install hint rather than a
# cryptic "command not found". Used as order-only prerequisites (`|`) so
# they don't force file targets to rebuild.

require-golangci-lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { \
		echo "Error: $(GOLANGCI_LINT) not found in PATH."; \
		echo "  brew install golangci-lint"; \
		echo "  or see https://golangci-lint.run/welcome/install/"; \
		exit 1; \
	}

require-gcov2lcov:
	@command -v $(GCOV2LCOV) >/dev/null 2>&1 || { \
		echo "Error: $(GCOV2LCOV) not found in PATH."; \
		echo "  go install github.com/jandelgado/gcov2lcov@latest"; \
		exit 1; \
	}

require-lcov:
	@command -v $(LCOV) >/dev/null 2>&1 || { \
		echo "Error: $(LCOV) not found in PATH."; \
		echo "  brew install lcov"; \
		echo "  or apt-get install lcov"; \
		exit 1; \
	}

require-genhtml:
	@command -v $(GENHTML) >/dev/null 2>&1 || { \
		echo "Error: $(GENHTML) not found in PATH (ships with lcov)."; \
		echo "  brew install lcov"; \
		echo "  or apt-get install lcov"; \
		exit 1; \
	}
