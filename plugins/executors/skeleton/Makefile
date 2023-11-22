PLUGINS = 

# Requires the installation of
#
# $ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
# $ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
idl: ../executor.proto ../supervisor.proto
	@echo "  Building protos"
	@ if ! which protoc > /dev/null; then \
		echo "error: protoc not installed" >&2; \
		exit 1; \
	fi	
	@protoc --proto_path=.. --go-grpc_out=. --go_out=. status.proto 	
	@protoc --proto_path=.. --go-grpc_out=. --go_out=. executor.proto 
	@protoc --proto_path=.. --go-grpc_out=. --go_out=. supervisor.proto 

build: idl
	@echo "  Building skeleton executor"
	@go build -o ../bin/bacalhau-exec-skeleton .

clean: 
	@echo "  Executors: Nothing to clean"	