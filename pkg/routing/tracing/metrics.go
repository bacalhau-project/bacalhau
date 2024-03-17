package tracing

import (
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.GetMeterProvider().Meter("node_info_store")

	addNodeDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"add_node",
		metric.WithDescription("Length of time taken to add a node"),
		metric.WithUnit("ms"),
	))
	listNodesDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"list_nodes",
		metric.WithDescription("Length of time taken to list all nodes"),
		metric.WithUnit("ms"),
	))
	getNodeDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"get_node",
		metric.WithDescription("Length of time taken to get a node"),
		metric.WithUnit("ms"),
	))
	getPrefixNodeDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"get_node_prefix",
		metric.WithDescription("Length of time taken to get a node by prefix"),
		metric.WithUnit("ms"),
	))
	deleteNodeDurationMilliseconds = lo.Must(meter.Int64Histogram(
		"delete_node",
		metric.WithDescription("Length of time taken to delete and purge a node"),
		metric.WithUnit("ms"),
	))
)
