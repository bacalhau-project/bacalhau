package wasm

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	wasmExecutorMeter = otel.GetMeterProvider().Meter("wasm-executor")
)

var (
	ActiveExecutions = lo.Must(telemetry.NewGauge(
		wasmExecutorMeter,
		"wasm_active_executions",
		"Number of active WASM executions",
	))
)
