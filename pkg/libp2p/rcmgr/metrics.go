package rcmgr

import (
	"context"
	"strings"

	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
)

// metricsReporter will build create a rcmgr.TraceReporter similar to the obs.StatsTraceReporter but with OpenTelemetry
// support rather than Prometheus.
func metricsReporter(meterProvider metric.MeterProvider) (rcmgr.TraceReporter, error) { //nolint:funlen
	meter := meterProvider.Meter("libp2p")

	connections, err := meter.Float64Counter(
		"rcmgr_connections",
		instrument.WithDescription("Number of Connections"),
	)
	if err != nil {
		return nil, err
	}
	peerConnections, err := meter.Float64Histogram(
		"rcmgr_peer_connections",
		instrument.WithDescription("Number of connections this peer has"),
	)
	if err != nil {
		return nil, err
	}
	previousPeerConnections, err := meter.Float64Histogram(
		"rcmgr_previous_peer_connections",
		instrument.WithDescription("Number of connections this peer previously had. "+
			"This is used to get the current connection number per peer histogram by subtracting this from the peer_connections histogram"),
	)
	if err != nil {
		return nil, err
	}

	streams, err := meter.Float64Counter(
		"rcmgr_streams",
		instrument.WithDescription("Number of Streams"),
	)
	if err != nil {
		return nil, err
	}
	peerStreams, err := meter.Float64Histogram(
		"rcmgr_peer_streams",
		instrument.WithDescription("Number of streams this peer has"),
	)
	if err != nil {
		return nil, err
	}
	previousPeerStreams, err := meter.Float64Histogram(
		"rcmgr_previous_peer_streams",
		instrument.WithDescription("Number of streams this peer has"),
	)
	if err != nil {
		return nil, err
	}

	memory, err := meter.Float64Counter(
		"rcmgr_memory",
		instrument.WithDescription("Amount of memory reserved as reported to the Resource Manager"),
	)
	if err != nil {
		return nil, err
	}
	peerMemory, err := meter.Float64Histogram(
		"rcmgr_peer_memory",
		instrument.WithDescription("How many peers have reserved this bucket of memory, as reported to the Resource Manager"),
	)
	if err != nil {
		return nil, err
	}
	previousPeerMemory, err := meter.Float64Histogram(
		"rcmgr_previous_peer_memory",
		instrument.WithDescription("How many peers have previously reserved this bucket of memory, as reported to the Resource Manager"),
	)
	if err != nil {
		return nil, err
	}

	connectionMemory, err := meter.Float64Histogram(
		"rcmgr_connection_memory",
		instrument.WithDescription("How many connections have reserved this bucket of memory, as reported to the Resource Manager"),
	)
	if err != nil {
		return nil, err
	}
	previousConnectionMemory, err := meter.Float64Histogram(
		"rcmgr_previous_connection_memory",
		instrument.WithDescription("How many connections have previously reserved this bucket of memory, as reported to the Resource Manager"),
	)
	if err != nil {
		return nil, err
	}

	fileDescriptors, err := meter.Float64Counter(
		"rcmgr_fds",
		instrument.WithDescription("Number of file descriptors reserved as reported to the Resource Manager"),
	)
	if err != nil {
		return nil, err
	}
	blocked, err := meter.Float64Counter(
		"rcmgr_blocked_resources",
		instrument.WithDescription("Number of blocked resources"),
	)
	if err != nil {
		return nil, err
	}

	return reporter{
		connections:              connections,
		peerConnections:          peerConnections,
		previousPeerConnections:  previousPeerConnections,
		streams:                  streams,
		peerStreams:              peerStreams,
		previousPeerStreams:      previousPeerStreams,
		memory:                   memory,
		peerMemory:               peerMemory,
		previousPeerMemory:       previousPeerMemory,
		connectionMemory:         connectionMemory,
		previousConnectionMemory: previousConnectionMemory,
		fileDescriptors:          fileDescriptors,
		blockedResources:         blocked,
	}, nil
}

var _ rcmgr.TraceReporter = reporter{}

type reporter struct {
	connections             instrument.Float64Counter
	peerConnections         instrument.Float64Histogram
	previousPeerConnections instrument.Float64Histogram

	streams             instrument.Float64Counter
	peerStreams         instrument.Float64Histogram
	previousPeerStreams instrument.Float64Histogram

	memory             instrument.Float64Counter
	peerMemory         instrument.Float64Histogram
	previousPeerMemory instrument.Float64Histogram

	connectionMemory         instrument.Float64Histogram
	previousConnectionMemory instrument.Float64Histogram

	fileDescriptors  instrument.Float64Counter
	blockedResources instrument.Float64Counter
}

// ConsumeEvent is a reimplementation of consumeEventWithLabelSlice in obs.StatsTraceReporter but using OTEL rather
// than Prometheus. Comments, variable names, and logic are all from the original with the only difference being how
// the metrics are recorded.
func (r reporter) ConsumeEvent(evt rcmgr.TraceEvt) { //nolint:funlen,gocyclo
	ctx := context.Background()

	switch evt.Type {
	case rcmgr.TraceAddStreamEvt, rcmgr.TraceRemoveStreamEvt:
		if p := rcmgr.PeerStrInScopeName(evt.Name); p != "" {
			// Aggregated peer stats. Counts how many peers have N number of streams open.
			// Uses two buckets aggregations. One to count how many streams the
			// peer has now. The other to count the negative value, or how many
			// streams did the peer use to have. When looking at the data you
			// take the difference from the two.

			oldStreamsOut := int64(evt.StreamsOut - evt.DeltaOut)
			peerStreamsOut := int64(evt.StreamsOut)
			if oldStreamsOut != peerStreamsOut {
				if oldStreamsOut != 0 {
					r.previousPeerStreams.Record(ctx, float64(oldStreamsOut), attribute.String("dir", "inbound"))
				}
				if peerStreamsOut != 0 {
					r.peerStreams.Record(ctx, float64(peerStreamsOut), attribute.String("dir", "outbound"))
				}
			}

			oldStreamsIn := int64(evt.StreamsIn - evt.DeltaIn)
			peerStreamsIn := int64(evt.StreamsIn)
			if oldStreamsIn != peerStreamsIn {
				if oldStreamsIn != 0 {
					r.previousPeerStreams.Record(ctx, float64(peerStreamsIn), attribute.String("dir", "inbound"))
				}
				if peerStreamsIn != 0 {
					r.peerStreams.Record(ctx, float64(peerStreamsIn), attribute.String("dir", "inbound"))
				}
			}
		} else {
			if evt.DeltaOut != 0 {
				if rcmgr.IsSystemScope(evt.Name) || rcmgr.IsTransientScope(evt.Name) {
					r.streams.Add(ctx, float64(evt.StreamsOut),
						attribute.String("dir", "outbound"),
						attribute.String("scope", evt.Name),
						attribute.String("protocol", ""),
					)
				} else if proto := rcmgr.ParseProtocolScopeName(evt.Name); proto != "" {
					r.streams.Add(ctx, float64(evt.StreamsOut),
						attribute.String("dir", "outbound"),
						attribute.String("scope", "protocol"),
						attribute.String("protocol", proto),
					)
				} else {
					// Not measuring service scope, connscope, servicepeer and protocolpeer. Lots of data, and
					// you can use aggregated peer stats + service stats to infer
					// this.
					break
				}
			}

			if evt.DeltaIn != 0 {
				if rcmgr.IsSystemScope(evt.Name) || rcmgr.IsTransientScope(evt.Name) {
					r.streams.Add(ctx, float64(evt.StreamsIn),
						attribute.String("dir", "inbound"),
						attribute.String("scope", evt.Name),
						attribute.String("protocol", ""),
					)
				} else if proto := rcmgr.ParseProtocolScopeName(evt.Name); proto != "" {
					r.streams.Add(ctx, float64(evt.StreamsIn),
						attribute.String("dir", "inbound"),
						attribute.String("scope", "protocol"),
						attribute.String("protocol", proto),
					)
				} else {
					// Not measuring service scope, connscope, servicepeer and protocolpeer. Lots of data, and
					// you can use aggregated peer stats + service stats to infer
					// this.
					break
				}
			}
		}
	case rcmgr.TraceAddConnEvt, rcmgr.TraceRemoveConnEvt:
		if p := rcmgr.PeerStrInScopeName(evt.Name); p != "" {
			// Aggregated peer stats. Counts how many peers have N number of connections.
			// Uses two buckets aggregations. One to count how many streams the
			// peer has now. The other to count the negative value, or how many
			// conns did the peer use to have. When looking at the data you
			// take the difference from the two.

			oldConnsOut := int64(evt.ConnsOut - evt.DeltaOut)
			connsOut := int64(evt.ConnsOut)
			if oldConnsOut != connsOut {
				if oldConnsOut != 0 {
					r.previousPeerConnections.Record(ctx, float64(oldConnsOut), attribute.String("dir", "outbound"))
				}
				if connsOut != 0 {
					r.peerConnections.Record(ctx, float64(oldConnsOut), attribute.String("dir", "outbound"))
				}
			}

			oldConnsIn := int64(evt.ConnsIn - evt.DeltaIn)
			connsIn := int64(evt.ConnsIn)
			if oldConnsIn != connsIn {
				if oldConnsIn != 0 {
					r.previousPeerConnections.Record(ctx, float64(oldConnsIn), attribute.String("dir", "inbound"))
				}
				if connsIn != 0 {
					r.peerConnections.Record(ctx, float64(connsIn), attribute.String("dir", "inbound"))
				}
			}
		} else {
			if rcmgr.IsConnScope(evt.Name) {
				// Not measuring this. I don't think it's useful.
				break
			}

			if rcmgr.IsSystemScope(evt.Name) {
				r.connections.Add(ctx, float64(evt.ConnsIn), attribute.String("dir", "inbound"), attribute.String("scope", "system"))
				r.connections.Add(ctx, float64(evt.ConnsOut), attribute.String("dir", "outbound"), attribute.String("scope", "system"))
			} else if rcmgr.IsTransientScope(evt.Name) {
				r.connections.Add(ctx, float64(evt.ConnsIn), attribute.String("dir", "inbound"), attribute.String("scope", "transient"))
				r.connections.Add(ctx, float64(evt.ConnsOut), attribute.String("dir", "outbound"), attribute.String("scope", "transient"))
			}

			// Represents the delta in fds
			if evt.Delta != 0 {
				if rcmgr.IsSystemScope(evt.Name) {
					r.fileDescriptors.Add(ctx, float64(evt.FD), attribute.String("scope", "system"))
				} else if rcmgr.IsTransientScope(evt.Name) {
					r.fileDescriptors.Add(ctx, float64(evt.FD), attribute.String("scope", "transient"))
				}
			}
		}
	case rcmgr.TraceReserveMemoryEvt, rcmgr.TraceReleaseMemoryEvt:
		if p := rcmgr.PeerStrInScopeName(evt.Name); p != "" {
			oldMem := evt.Memory - evt.Delta
			if oldMem != evt.Memory {
				if oldMem != 0 {
					r.previousPeerMemory.Record(ctx, float64(oldMem))
				}
				if evt.Memory != 0 {
					r.peerMemory.Record(ctx, float64(evt.Memory))
				}
			}
		} else if rcmgr.IsConnScope(evt.Name) {
			oldMem := evt.Memory - evt.Delta
			if oldMem != evt.Memory {
				if oldMem != 0 {
					r.previousConnectionMemory.Record(ctx, float64(oldMem))
				}
				if evt.Memory != 0 {
					r.connectionMemory.Record(ctx, float64(evt.Memory))
				}
			}
		} else {
			if rcmgr.IsSystemScope(evt.Name) || rcmgr.IsTransientScope(evt.Name) {
				r.memory.Add(ctx, float64(evt.Memory), attribute.String("scope", evt.Name), attribute.String("protocol", ""))
			} else if proto := rcmgr.ParseProtocolScopeName(evt.Name); proto != "" {
				r.memory.Add(ctx, float64(evt.Memory), attribute.String("scope", "protocol"), attribute.String("protocol", proto))
			} else {
				// Not measuring connscope, servicepeer and protocolpeer. Lots of data, and
				// you can use aggregated peer stats + service stats to infer
				// this.
				break
			}
		}

	case rcmgr.TraceBlockAddConnEvt, rcmgr.TraceBlockAddStreamEvt, rcmgr.TraceBlockReserveMemoryEvt:
		var resource string
		if evt.Type == rcmgr.TraceBlockAddConnEvt {
			resource = "connection"
		} else if evt.Type == rcmgr.TraceBlockAddStreamEvt {
			resource = "stream"
		} else {
			resource = "memory"
		}

		scopeName := evt.Name
		// Only the top scopeName. We don't want to get the peerid here.
		// Using indexes and slices to avoid allocating.
		scopeSplitIdx := strings.IndexByte(scopeName, ':')
		if scopeSplitIdx != -1 {
			scopeName = evt.Name[0:scopeSplitIdx]
		}
		// Drop the connection or stream id
		idSplitIdx := strings.IndexByte(scopeName, '-')
		if idSplitIdx != -1 {
			scopeName = scopeName[0:idSplitIdx]
		}

		if evt.DeltaIn != 0 {
			r.blockedResources.Add(ctx, float64(evt.DeltaIn),
				attribute.String("dir", "inbound"),
				attribute.String("scope", scopeName),
				attribute.String("resource", resource),
			)
		}

		if evt.DeltaOut != 0 {
			r.blockedResources.Add(ctx, float64(evt.DeltaOut),
				attribute.String("dir", "outbound"),
				attribute.String("scope", scopeName),
				attribute.String("resource", resource),
			)
		}

		if evt.Delta != 0 && resource == "connection" {
			// This represents fds blocked
			r.blockedResources.Add(ctx, float64(evt.Delta),
				attribute.String("dir", ""),
				attribute.String("scope", scopeName),
				attribute.String("resource", "fd"),
			)
		} else if evt.Delta != 0 {
			r.blockedResources.Add(ctx, float64(evt.Delta),
				attribute.String("dir", ""),
				attribute.String("scope", scopeName),
				attribute.String("resource", resource),
			)
		}
	}
}
