package requester

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	requesterMeter = otel.GetMeterProvider().Meter("requester")
)

var (
	JobsSubmitted = lo.Must(telemetry.NewCounter(
		requesterMeter,
		"job_submitted",
		"Number of jobs submitted"))
)
