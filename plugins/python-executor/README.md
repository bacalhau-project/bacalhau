# Bacalhau Python Executor Plugin

This program provides a pluggable executor for [Bacalhau](https://bacalhau.org), in particular this executor allows for the execution of Python scripts/programs. 

This executor currently only binds to 127.0.0.1, and port 2112.
You can change the port that the executor listens on by using an environment variable:

```
export PYTHON_EXECUTOR_PORT=22111
```

## Building the plugin

* Install Rust (>1.70) either from your usual package manager, or from [Rustup](https://rustup.rs/).
* To build for debug, use `cargo build`.
* To build for production, use `cargo build --release`. 

Should you wish to run tests, use `cargo test`.


## Invoking the server manually

Although there is not yet a bacalhau test client, you can use `grpcurl` to interact with the server.

First, run the server with `cargo run`, then 

### List all services 

```shell
grpcurl -plaintext localhost:2112 list
```

### Describe a specific service interface 

```shell
grpcurl -plaintext localhost:2112 describe executor.Executor
```

### Describe a specific function 

```shell
grpcurl -plaintext localhost:2112 describe executor.Executor.Run
```

# Call Run 

Encodes the json {"ExecutionID":"123"} as base64 to call Run,
and then extract the result and de-base64 for output. 

```shell
grpcurl -plaintext -proto ./proto/executor.proto \
 -d '{"Params": "eyJFeGVjdXRpb25JRCI6IjEyMyJ9Cg=="}' \
 127.0.0.1:2112 \
 executor.Executor.Run | jq -r '.Params' | base64 -D 
``````

## Caveats 

This project currently has its own copy of the protobuf files used by the other executors. It should maintain a link to the shared protobufs. 