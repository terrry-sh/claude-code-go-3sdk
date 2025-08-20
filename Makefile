.PHONY: help test build clean fmt lint install-deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

install-deps: ## Install dependencies
	go mod download
	go mod tidy

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

build: ## Build the library
	go build -v ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

lint: ## Run linters
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping linting"; \
	fi

clean: ## Clean build artifacts
	go clean
	rm -f coverage.out coverage.html

example-quick: ## Run quick start example
	go run examples/quick_start.go

example-streaming: ## Run streaming mode example (specify target with TARGET=)
	@if [ -z "$(TARGET)" ]; then \
		go run examples/streaming_mode.go; \
	else \
		go run examples/streaming_mode.go $(TARGET); \
	fi

verify: fmt lint test ## Run all verification steps
	@echo "All verification steps passed!"