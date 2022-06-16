package executor

type ExecutorType string

const EXECUTOR_DOCKER ExecutorType = "docker"
const EXECUTOR_NOOP ExecutorType = "noop"
const EXECUTOR_WASM ExecutorType = "wasm"

var EXECUTORS = []string{
	string(EXECUTOR_DOCKER),
	string(EXECUTOR_NOOP),
	string(EXECUTOR_WASM),
}
