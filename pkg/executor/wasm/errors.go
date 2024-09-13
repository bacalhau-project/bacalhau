package wasm

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const WasmExecutorComponent = "Executor/Wasm"

func NewWasmExecutorError(code models.ErrorCode, message string) *models.BaseError {
	return models.NewBaseError(message).
		WithCode(code).
		WithComponent(WasmExecutorComponent)
}
