module github.com/bacalhau-project/bacalhau/testdata/wasm/http/src

go 1.23

toolchain go1.23.0

require github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client v0.0.0

replace github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/client => ../../../../pkg/executor/wasm/funcs/http/client
