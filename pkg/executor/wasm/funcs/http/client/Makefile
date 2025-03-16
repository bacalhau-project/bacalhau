.PHONY: all check clean help

# Default target
all: check

# Check if the code compiles with TinyGo and format/lint the code
check:
	@echo "Checking if the client compiles with TinyGo..."
	@mkdir -p build
	@tinygo build -o build/client.wasm -target wasi -no-debug .
	@echo "Formatting code..."
	@gofmt -w *.go
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
	@echo "  all     - Default target, runs check"
	@echo "  check   - Check if the client compiles with TinyGo, format and lint code"
	@echo "  clean   - Clean build artifacts"
	@echo "  help    - Show this help message" 