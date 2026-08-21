# Makefile for go-lua-parser
#
# Local dev commands mirror .github/workflows/ci.yml so `make lint` and
# `make test` verify the same things CI does. The coverage targets mirror
# .github/workflows/badge.yaml (LCOV via gcov2lcov + lcov/genhtml).

GO             ?= go
GOLANGCI_LINT  ?= golangci-lint
GCOV2LCOV      ?= gcov2lcov
LCOV           ?= lcov
GENHTML        ?= genhtml

COVERAGE_OUT   := coverage.out
LCOV_DIR       := coverage
LCOV_FILE      := $(LCOV_DIR)/lcov.info
HTML_DIR       := htmlcov

.PHONY: help lint test test-coverage test-html vet verify build clean \
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
	@echo "  make test-coverage   - Report LCOV line coverage (matches CI gate)"
	@echo "  make test-html       - Generate HTML coverage report under $(HTML_DIR)/"
	@echo ""
	@echo "Build:"
	@echo "  make build           - Build all packages"
	@echo ""
	@echo "Housekeeping:"
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

test:
	@$(GO) test -race -count=1 -coverprofile=$(COVERAGE_OUT) ./...

# Compute the same line-coverage percentage CI gates on (see ci.yml). Local
# runs of this target need lcov and gcov2lcov; `make test` alone does not.
test-coverage: test require-gcov2lcov require-lcov
	@mkdir -p $(LCOV_DIR)
	@$(GCOV2LCOV) -infile=$(COVERAGE_OUT) -outfile=$(LCOV_FILE)
	@COVERAGE=$$($(LCOV) --summary $(LCOV_FILE) 2>&1 | grep -i '^\s*lines' | grep -oE '[0-9]+\.[0-9]+' | head -1); \
		echo "Line coverage: $${COVERAGE}%"

# Local HTML report — same generator (genhtml) that produces the deployed
# report at https://jedi-knights.github.io/go-lua-parser/.
test-html: test-coverage require-genhtml
	@$(GENHTML) $(LCOV_FILE) -o $(HTML_DIR)/ --quiet --ignore-errors source
	@echo "Report: $(HTML_DIR)/index.html"

clean:
	@rm -rf $(COVERAGE_OUT) $(LCOV_DIR) $(HTML_DIR) test-results.json

# Tool prerequisites — fail with an actionable install hint rather than a
# cryptic "command not found".

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
