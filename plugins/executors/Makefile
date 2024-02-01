PLUGINS = 
TMP_OUT = ../../

# As this is building Go specific libraries, it is expected
# that this will live inside each PLUGINS which is Go-based,
# but is implemented here to allow us to iteratively move the 
# existing plugins. Requires the installation of
#
# $ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
# $ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
#
# We should move this task to the individual plugins that require
# it once they exist.
idl: executor.proto
	@echo "  Building protos"
	@ if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi	
	protoc --proto_path=. --go-grpc_out=$(TMP_OUT) --go_out=$(TMP_OUT)  executor.proto 

build: idl
	@echo "  No plugins to build yet"

clean: 
	@echo "  Executors: Nothing to clean"	