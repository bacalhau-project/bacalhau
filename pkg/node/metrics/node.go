package metrics

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"

	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
)

var (
	nodeMeter = otel.GetMeterProvider().Meter("bacalhau-node")
)

var (
	NodeInfo = lo.Must(telemetry.NewCounter(
		nodeMeter,
		"bacalhau_node_info",
		"A static metric with labels describing the bacalhau node",
	))
)
