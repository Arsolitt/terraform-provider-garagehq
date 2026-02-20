.PHONY: help build install test lint fmt lint-only clean release-dry-run

# Docker image versions
GOLANGCI_LINT_VERSION := v2.10.1

# Provider version - use git tag or fallback to "dev"
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")

# Provider registry path
PROVIDER_REGISTRY := ~/.terraform.d/plugins/registry.terraform.io/arsolitt/garagehq/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)

# Default target
help:
	@echo "Available targets:"
	@echo "  build          - Build the terraform provider"
	@echo "  install        - Build and install the provider locally"
	@echo "  test           - Run tests with coverage"
	@echo "  lint           - Format code and run golangci-lint"
	@echo "  fmt            - Format code using golangci-lint"
	@echo "  lint-only      - Run golangci-lint without formatting"
	@echo "  clean          - Clean build artifacts"
	@echo "  release-dry-run - Test release build without publishing"
	@echo ""
	@echo "Version: $(VERSION)"

build:
	go build -o terraform-provider-garagehq

install: build
	@echo "Installing provider version $(VERSION)..."
	mkdir -p $(PROVIDER_REGISTRY)
	cp terraform-provider-garagehq $(PROVIDER_REGISTRY)/
	@echo "Installed to $(PROVIDER_REGISTRY)"

test:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./... || true

# Format code using golangci-lint formatters (faster than separate tools)
fmt:
	docker run --rm \
		-u "$(shell id -u):$(shell id -g)" \
		-e GOCACHE=/tmp/go-cache \
		-e GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache \
		-v "$(PWD):/app" \
		-v "$(HOME)/.cache:/home/cache" \
		-w /app \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
		golangci-lint run --fix

# Run golangci-lint (formats first, then lints)
lint:
	docker run --rm \
		-u "$(shell id -u):$(shell id -g)" \
		-e GOCACHE=/tmp/go-cache \
		-e GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache \
		-v "$(PWD):/app" \
		-v "$(HOME)/.cache:/home/cache" \
		-w /app \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
		golangci-lint run --fix

# Run only linting without formatting
lint-only:
	docker run --rm \
		-u "$(shell id -u):$(shell id -g)" \
		-e GOCACHE=/tmp/go-cache \
		-e GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache \
		-v "$(PWD):/app" \
		-v "$(HOME)/.cache:/home/cache" \
		-w /app \
		golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) \
		golangci-lint run

# Test release build locally (requires goreleaser)
release-dry-run:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not found. Install from https://goreleaser.com/"; exit 1; }
	goreleaser release --snapshot --clean

clean:
	go clean
	rm -f terraform-provider-garagehq coverage.txt
