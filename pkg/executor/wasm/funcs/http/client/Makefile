.PHONY: all check fmt lint clean help

# Default target
all: check fmt lint

# Check if the code compiles with TinyGo
check:
	@echo "Checking if the client compiles with TinyGo..."
	@mkdir -p build
	@tinygo build -o build/client.wasm -target wasi -no-debug .

# Format the code
fmt:
	@echo "Formatting code..."
	@gofmt -w *.go

# Lint the code
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Skipping linting."; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build

# Help target
help:
	@echo "Available targets:"
	@echo "  all     - Default target, runs check, fmt, and lint"
	@echo "  check   - Check if the client compiles with TinyGo"
	@echo "  fmt     - Format the code"
	@echo "  lint    - Lint the code"
	@echo "  clean   - Clean build artifacts"
	@echo "  help    - Show this help message" 