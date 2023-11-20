
BIN = ../bin/bacalhau-exec-wasm

idl: ../executor.proto
	@echo "  Building protobuf"
	@if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi	
	@protoc --proto_path=.. \
		--go_out=. \
		--go-grpc_out=. \
		../executor.proto 
	
build: idl 
	@echo "  Building binary"
	@go build -o $(BIN) . 

run: idl 
	@echo "  Build + Run binary"
	@go run .

clean:
	@echo "  Cleaning binary"
	@rm $(BIN)