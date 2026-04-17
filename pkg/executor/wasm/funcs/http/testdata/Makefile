.PHONY: all clean force

all: main.wasm

main.wasm: src/main.go src/go.mod
	@echo "Building HTTP test module..."
	@cd src && tinygo build -o ../main.wasm -target wasi .
	@echo "Build completed successfully!"

force:
	@echo "Force rebuilding HTTP test module..."
	@cd src && tinygo build -o ../main.wasm -target wasi .
	@echo "Build completed successfully!"

clean:
	@echo "Cleaning HTTP test module..."
	@rm -f main.wasm 