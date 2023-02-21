package tracing

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/telemetry"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func NewTracedClient() (TracedClient, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return TracedClient{}, err
	}

	return TracedClient{
		client:   c,
		hostname: c.DaemonHost(),
	}, nil
}

// TracedClient is a client for Docker that also traces requests being made to it. The names of traces are inspired by
// the docker CLI, so ContainerRemove becomes `docker.container.rm` which would be `docker container rm` on the
// command line.
type TracedClient struct {
	client   *client.Client
	hostname string
}

func (c TracedClient) ContainerCreate(
	ctx context.Context,
	config *container.Config,
	hostConfig *container.HostConfig,
	networkingConfig *network.NetworkingConfig,
	platform *v1.Platform,
	name string,
) (container.ContainerCreateCreatedBody, error) {
	ctx, span := c.span(ctx, "container.create")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[container.ContainerCreateCreatedBody](span)(
		c.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, name),
	)
}

func (c TracedClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	ctx, span := c.span(ctx, "container.inspect")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[types.ContainerJSON](span)(c.client.ContainerInspect(ctx, containerID))
}

func (c TracedClient) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	ctx, span := c.span(ctx, "container.list")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[[]types.Container](span)(c.client.ContainerList(ctx, options))
}

func (c TracedClient) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	ctx, span := c.span(ctx, "container.logs")
	// span ends when the io.ReadCloser is closed

	return telemetry.RecordErrorOnSpanReadCloserAndClose(span)(c.client.ContainerLogs(ctx, container, options))
}

func (c TracedClient) ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	ctx, span := c.span(ctx, "container.rm")
	defer span.End()

	return telemetry.RecordErrorOnSpan(span)(c.client.ContainerRemove(ctx, containerID, options))
}

func (c TracedClient) ContainerStart(ctx context.Context, id string, options types.ContainerStartOptions) error {
	ctx, span := c.span(ctx, "container.start")
	defer span.End()

	return telemetry.RecordErrorOnSpan(span)(c.client.ContainerStart(ctx, id, options))
}

func (c TracedClient) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	ctx, span := c.span(ctx, "container.stop")
	defer span.End()

	return telemetry.RecordErrorOnSpan(span)(c.client.ContainerStop(ctx, containerID, timeout))
}

func (c TracedClient) ContainerWait(
	ctx context.Context,
	containerID string,
	condition container.WaitCondition,
) (<-chan container.ContainerWaitOKBody, <-chan error) {
	ctx, span := c.span(ctx, "container.wait")
	// span ends when one of the channels sends a message

	return telemetry.RecordErrorOnSpanTwoChannels[container.ContainerWaitOKBody](span)(c.client.ContainerWait(ctx, containerID, condition))
}

func (c TracedClient) CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	ctx, span := c.span(ctx, "container.cp")
	// span ends when the io.ReadCloser is closed

	return telemetry.RecordErrorOnSpanReadCloserTwoAndClose[types.ContainerPathStat](span)(
		c.client.CopyFromContainer(ctx, containerID, srcPath),
	)
}

func (c TracedClient) ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error) {
	ctx, span := c.span(ctx, "image.inspect")
	defer span.End()

	return telemetry.RecordErrorOnSpanThree[types.ImageInspect, []byte](span)(c.client.ImageInspectWithRaw(ctx, imageID))
}

func (c TracedClient) ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error) {
	ctx, span := c.span(ctx, "image.pull")
	// span ends when the io.ReadCloser is closed

	// The span won't be annotated with _all_ possible failures as the error returned only covers immediate errors.
	// Errors occurring while pulling the image are returned within the io.ReadCloser and currently not captured.
	return telemetry.RecordErrorOnSpanReadCloserAndClose(span)(c.client.ImagePull(ctx, refStr, options))
}

func (c TracedClient) NetworkConnect(ctx context.Context, networkID, containerID string, config *network.EndpointSettings) error {
	ctx, span := c.span(ctx, "network.connect")
	defer span.End()

	return telemetry.RecordErrorOnSpan(span)(c.client.NetworkConnect(ctx, networkID, containerID, config))
}

func (c TracedClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	ctx, span := c.span(ctx, "network.create")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[types.NetworkCreateResponse](span)(c.client.NetworkCreate(ctx, name, options))
}

func (c TracedClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	ctx, span := c.span(ctx, "network.list")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[[]types.NetworkResource](span)(c.client.NetworkList(ctx, options))
}

func (c TracedClient) NetworkInspect(
	ctx context.Context,
	networkID string,
	options types.NetworkInspectOptions,
) (types.NetworkResource, error) {
	ctx, span := c.span(ctx, "network.inspect")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[types.NetworkResource](span)(c.client.NetworkInspect(ctx, networkID, options))
}

func (c TracedClient) NetworkRemove(ctx context.Context, networkID string) error {
	ctx, span := c.span(ctx, "network.rm")
	defer span.End()

	return telemetry.RecordErrorOnSpan(span)(c.client.NetworkRemove(ctx, networkID))
}

func (c TracedClient) Info(ctx context.Context) (types.Info, error) {
	ctx, span := c.span(ctx, "info")
	defer span.End()

	return telemetry.RecordErrorOnSpanTwo[types.Info](span)(c.client.Info(ctx))
}

func (c TracedClient) Close() error {
	return c.client.Close()
}

func (c TracedClient) span(ctx context.Context, name string) (context.Context, trace.Span) {
	return system.NewSpan(
		ctx,
		system.GetTracer(),
		fmt.Sprintf("docker.%s", name),
		trace.WithAttributes(semconv.HostName(c.hostname), semconv.PeerService("docker")),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}
