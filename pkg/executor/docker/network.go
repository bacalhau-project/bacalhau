package docker

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	dockerNetworkNone   = container.NetworkMode("none")
	dockerNetworkHost   = container.NetworkMode("host")
	dockerNetworkBridge = container.NetworkMode("bridge")
)

const (
	// The Docker image used to provide HTTP filtering and throttling. See
	// pkg/executor/docker/gateway/Dockerfile for design notes. We specify this
	// using a fully-versioned tag so that the interface between code and image
	// stay in sync.
	httpGatewayImage = "ghcr.io/bacalhau-project/http-gateway:v0.3.16"

	// The hostname used by Mac OS X and Windows hosts to refer to the Docker
	// host in a network context. Linux hosts can use this hostname if they
	// are set up using the `dockerHostAddCommand` as an extra host.
	dockerHostHostname = "host.docker.internal"

	// The magic word recognized by the Docker engine in place of an IP address
	// that always maps to the IP address of the Docker host.
	dockerHostIPAddressMagicWord = "host-gateway"

	// A string that can be passed as an ExtraHost to a Docker run or create
	// command that will ensure the host is visible on the network from within
	// the container, even on a Linux host where localhost is sufficient.
	dockerHostAddCommand = dockerHostHostname + ":" + dockerHostIPAddressMagicWord
)

const (
	// The port used by the proxy server within the HTTP gateway container. This
	// is also specified in squid.conf and gateway.sh.
	httpProxyPort = 8080
)

var (
	// The capabilities that the gateway container needs. See the Dockerfile.
	gatewayCapabilities = []string{"NET_ADMIN"}
)

func (e *Executor) setupNetworkForJob(
	ctx context.Context,
	shard model.JobShard,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
) (err error) {
	containerConfig.NetworkDisabled = shard.Job.Spec.Network.Disabled()
	switch shard.Job.Spec.Network.Type {
	case model.NetworkNone:
		hostConfig.NetworkMode = dockerNetworkNone
	case model.NetworkFull:
		hostConfig.NetworkMode = dockerNetworkHost
		hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, dockerHostAddCommand)
	case model.NetworkHTTP:
		var internalNetwork *types.NetworkResource
		var proxyAddr *net.TCPAddr
		internalNetwork, proxyAddr, err = e.createHTTPGateway(ctx, shard)
		if err != nil {
			return
		}
		hostConfig.NetworkMode = container.NetworkMode(internalNetwork.Name)
		containerConfig.Env = append(containerConfig.Env,
			fmt.Sprintf("http_proxy=%s", proxyAddr.String()),
			fmt.Sprintf("https_proxy=%s", proxyAddr.String()),
		)
	default:
		err = fmt.Errorf("unsupported network type %q", shard.Job.Spec.Network.Type.String())
	}
	return
}

func (e *Executor) createHTTPGateway(
	ctx context.Context,
	shard model.JobShard,
) (*types.NetworkResource, *net.TCPAddr, error) {
	// Create an internal only bridge network to join our gateway and job container
	networkResp, err := e.Client.NetworkCreate(ctx, e.dockerObjectName(shard, "network"), types.NetworkCreate{
		Driver:     "bridge",
		Scope:      "local",
		Internal:   true,
		Attachable: true,
		Labels:     e.jobContainerLabels(&shard),
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "error creating network")
	}

	// Get the subnet that Docker has picked for the newly created network
	internalNetwork, err := e.Client.NetworkInspect(ctx, networkResp.ID, types.NetworkInspectOptions{})
	if err != nil || len(internalNetwork.IPAM.Config) < 1 {
		return nil, nil, errors.Wrap(err, "error getting network subnet")
	}
	subnet := internalNetwork.IPAM.Config[0].Subnet

	// Create the gateway container initially attached to the *host* network
	domainList := strings.Join(shard.Job.Spec.Network.Domains, "\n")
	gatewayContainer, err := e.Client.ContainerCreate(ctx, &container.Config{
		Image: httpGatewayImage,
		Env: []string{
			fmt.Sprintf("BACALHAU_HTTP_CLIENTS=%s", subnet),
			fmt.Sprintf("BACALHAU_HTTP_DOMAINS=%s", domainList),
			fmt.Sprintf("BACALHAU_JOB_ID=%s", shard.Job.Metadata.ID),
		},
		Healthcheck:     &container.HealthConfig{}, //TODO
		NetworkDisabled: false,
		Labels:          e.jobContainerLabels(&shard),
	}, &container.HostConfig{
		NetworkMode: dockerNetworkBridge,
		CapAdd:      gatewayCapabilities,
		ExtraHosts:  []string{dockerHostAddCommand},
	}, nil, nil, e.dockerObjectName(shard, "gateway"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "error creating gateway container")
	}

	// Attach the bridge network to the container
	err = e.Client.NetworkConnect(ctx, internalNetwork.ID, gatewayContainer.ID, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error attaching network to gateway")
	}

	// Start the container and wait for it to come up
	err = e.Client.ContainerStart(ctx, gatewayContainer.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to start network gateway container")
	}

	stdout, stderr, err := docker.FollowLogs(ctx, e.Client, gatewayContainer.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get gateway container logs")
	}
	go logger.LogStream(log.Ctx(ctx).With().Str("Source", "stdout").Logger().WithContext(ctx), stdout)
	go logger.LogStream(log.Ctx(ctx).With().Str("Source", "stderr").Logger().WithContext(ctx), stderr)

	// Look up the IP address of the gateway container and attach it to the spec
	containerDetails, err := e.Client.ContainerInspect(ctx, gatewayContainer.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error getting gateway container details")
	}
	networkAttachment, ok := containerDetails.NetworkSettings.Networks[internalNetwork.Name]
	if !ok || networkAttachment.IPAddress == "" {
		return nil, nil, fmt.Errorf("gateway does not appear to be attached to internal network")
	}
	proxyIP := net.ParseIP(networkAttachment.IPAddress)
	proxyAddr := net.TCPAddr{IP: proxyIP, Port: httpProxyPort}
	return &internalNetwork, &proxyAddr, err
}
