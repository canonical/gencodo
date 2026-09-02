# SPDX-License-Identifier: LGPL-3.0-only
# Copyright 2025-2026 Canonical Ltd.

.PHONY: all build test test-verbose test-race test-coverage fmt vet lint reuse install deps tidy clean release proxy-warm help

GOLANGCI_LINT ?= golangci-lint
REUSE ?= reuse

# Default target
all: fmt vet lint test

# Build the project (this is a library, so we verify it compiles)
build:
	@echo "Building..."
	@go build -v ./...

# Run tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests verbosely
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run golangci-lint (configuration in .golangci.yaml)
lint:
	@echo "Running golangci-lint..."
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { \
		echo "golangci-lint not installed; run 'make install' or see https://golangci-lint.run/docs/welcome/install/"; \
		exit 1; }
	@$(GOLANGCI_LINT) run ./...

# Check REUSE (license and copyright) compliance
reuse:
	@echo "Running reuse lint..."
	@command -v $(REUSE) >/dev/null 2>&1 || { \
		echo "reuse not installed; run 'pipx install reuse' (or 'make REUSE=\"pipx run reuse\" reuse')"; \
		exit 1; }
	@$(REUSE) lint

# Install development tools
install:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@command -v pipx >/dev/null 2>&1 && pipx install reuse || echo "pipx not found; install reuse manually: https://reuse.software"
	@echo "Development tools installed successfully"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@go clean

# Release a new version (requires VERSION=vX.Y.Z and a matching CHANGELOG.md section).
# Pushing the tag triggers .github/workflows/release.yml, which publishes the
# GitHub release with the CHANGELOG section as its notes.
release:
ifndef VERSION
	@echo "Error: VERSION is required. Usage: make release VERSION=v0.3.0"
	@exit 1
endif
	@echo "Checking version format..."
	@if ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Error: VERSION must follow format vX.Y.Z (e.g., v0.3.0)"; \
		exit 1; \
	fi
	@echo "Checking CHANGELOG.md has a section for $(VERSION)..."
	@if ! grep -qE '^## \[$(VERSION:v%=%)\] - [0-9]{4}-[0-9]{2}-[0-9]{2}$$' CHANGELOG.md; then \
		echo "Error: CHANGELOG.md has no '## [$(VERSION:v%=%)] - YYYY-MM-DD' section"; \
		exit 1; \
	fi
	@echo "Checking working tree is clean..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: commit or stash your changes before releasing"; \
		exit 1; \
	fi
	@echo "Running checks before release..."
	@$(MAKE) vet test-race
	@echo "Creating tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Pushing tag $(VERSION) to origin..."
	@git push origin $(VERSION)
	@echo "Release $(VERSION) tagged; the release workflow publishes the GitHub release."
	@echo "Users can install it with: go get github.com/canonical/gencodo@$(VERSION)"

# Ask the Go module proxy to index a released version (optional; requires VERSION=vX.Y.Z)
proxy-warm:
ifndef VERSION
	@echo "Error: VERSION is required. Usage: make proxy-warm VERSION=v0.3.0"
	@exit 1
endif
	@GOPROXY=https://proxy.golang.org GOFLAGS=-mod=mod go list -m github.com/canonical/gencodo@$(VERSION)

# Show help
help:
	@echo "Available targets:"
	@echo "  all            - Format, vet, lint, and test (default)"
	@echo "  build          - Build the project"
	@echo "  test           - Run tests"
	@echo "  test-verbose   - Run tests verbosely"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  reuse          - Check REUSE license/copyright compliance"
	@echo "  install        - Install development tools (golangci-lint, reuse)"
	@echo "  deps           - Download dependencies"
	@echo "  tidy           - Tidy dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  release        - Tag and push a release (requires VERSION=vX.Y.Z and a CHANGELOG section)"
	@echo "  proxy-warm     - Ask proxy.golang.org to index a version (requires VERSION)"
	@echo "  help           - Show this help message"
