package docker

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	dockerExecutorMeter = otel.GetMeterProvider().Meter("docker-executor")
)

var (
	ActiveExecutions = lo.Must(telemetry.NewGauge(
		dockerExecutorMeter,
		"docker_active_executions",
		"Number of active docker executions",
	))
)
