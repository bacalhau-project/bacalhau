.PHONY: all clean

all: main.wasm

main.wasm: src/main.go src/client/client.go src/go.mod
	@echo "Building HTTP test module..."
	@CURR_DIR=$$(pwd); \
	TEMP_DIR=$$(mktemp -d -t http-test-XXXXXX); \
	cp -r src $$TEMP_DIR/; \
	cd $$TEMP_DIR/src && tinygo build -o ../main.wasm -target wasi .; \
	cp $$TEMP_DIR/main.wasm $$CURR_DIR/; \
	rm -rf $$TEMP_DIR; \
	echo "Build completed successfully!"

clean:
	@echo "Cleaning HTTP test module..."
	@rm -f main.wasm 