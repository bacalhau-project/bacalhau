module http-test

go 1.24

toolchain go1.24.0

require github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client v0.0.0

replace github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client => ../../client
