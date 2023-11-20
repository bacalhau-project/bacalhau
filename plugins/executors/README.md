
# Executor plugins 

Executors are responsible for running jobs in Bacalhau, and this directory is the home of the builtin Bacalhau plugins for Docker and WebAssembly jobs. 


## WIP: Building plugins 

To build the default plugins once they are implemented here, you can use the provided Makefile.
Before building the plugins you will need to build the protobufs, and this requires that you install
the following tools.

```sh
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

```sh 
# Build 
make build

# Cleanup
make clean 
```

This will run the build, and clean, tasks respectively for each plugin that lives in this directory.
