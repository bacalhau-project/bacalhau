package executor

type executorType string

const EXECUTOR_DOCKER executorType = "docker"
const EXECUTOR_NOOP executorType = "noop"
const EXECUTOR_WASM executorType = "wasm"

var EXECUTORS = []string{
	string(EXECUTOR_DOCKER),
	string(EXECUTOR_NOOP),
	string(EXECUTOR_WASM),
}
