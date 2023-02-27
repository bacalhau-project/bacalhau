package rcmgr

import (
	"bufio"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"golang.org/x/exp/slices"
	"io"
	"testing"
	"time"
)

func TestMetricsReporter(t *testing.T) {
	// Connect two libp2p hosts together, send a message, then make sure a metric was reported

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	exp, err := otlpmetricgrpc.New(ctx)
	require.NoError(t, err)
	reader := sdkmetric.NewPeriodicReader(exp)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	port1, err := freeport.GetFreePort()
	require.NoError(t, err)

	host1 := startListener(t, port1, meterProvider)

	port2, err := freeport.GetFreePort()
	require.NoError(t, err)

	host2 := startListener(t, port2, meterProvider)

	host2.SetStreamHandler("/echo/1.0", func(s network.Stream) {
		buf := bufio.NewReader(s)
		str, err := buf.ReadString('\n')
		if !assert.NoError(t, err) {
			assert.NoError(t, s.Reset())
			return
		}

		if _, err = s.Write([]byte(str)); !assert.NoError(t, err) {
			assert.NoError(t, s.Reset())
			return
		}
		assert.NoError(t, s.Close())
	})

	host1.Peerstore().AddAddrs(host2.ID(), host2.Addrs(), peerstore.PermanentAddrTTL)
	stream, err := host1.NewStream(ctx, host2.ID(), "/echo/1.0")
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, stream.Close())
	})

	_, err = stream.Write([]byte("testing\n"))
	require.NoError(t, err)

	read, err := io.ReadAll(stream)
	require.NoError(t, err)
	assert.Equal(t, "testing\n", string(read))

	metrics, err := reader.Collect(ctx)
	require.NoError(t, err)

	t.Log(metrics)

	require.Len(t, metrics.ScopeMetrics, 1)

	connectionsMetric := slices.IndexFunc(metrics.ScopeMetrics[0].Metrics, func(metrics metricdata.Metrics) bool {
		return metrics.Name == "rcmgr_connections"
	})
	require.NotEqual(t, -1, connectionsMetric)

	connectionData := metrics.ScopeMetrics[0].Metrics[connectionsMetric].Data.(metricdata.Sum[float64])
	outboundConnection := slices.IndexFunc(connectionData.DataPoints, func(d metricdata.DataPoint[float64]) bool {
		if bound, ok := d.Attributes.Value("dir"); !ok {
			return false
		} else if bound.AsString() != "outbound" {
			return false
		}

		if scope, ok := d.Attributes.Value("scope"); !ok {
			return false
		} else if scope.AsString() != "transient" {
			return false
		}

		return true
	})
	require.NotEqual(t, -1, outboundConnection)

	assert.NotZero(t, connectionData.DataPoints[outboundConnection].Value)
}

func startListener(t *testing.T, port int, provider metric.MeterProvider) host.Host {
	priv, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	require.NoError(t, err)

	t.Log("Starting listener on port", port)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
		libp2p.NoSecurity,
		resourceManagerWithMetricsProvider(provider),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, h.Close())
	})

	return h
}
