package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/docker/go-connections/nat"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
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
	httpGatewayImage = "ghcr.io/bacalhau-project/http-gateway:v0.3.17"

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

	// This time should match the --interval= option specified on the container
	// HEALTHCHECK (as the health status only updates this frequently so more
	// frequent calls are useless)
	httpGatewayHealthcheckInterval = time.Second

	// The port used by the proxy server within the HTTP gateway container. This
	// is also specified in squid.conf and gateway.sh.
	httpProxyPort = 8080
)

var (
	// The capabilities that the gateway container needs. See the Dockerfile.
	gatewayCapabilities = []string{"NET_ADMIN"}
)

//nolint:nakedret
func (e *Executor) setupNetworkForJob(
	ctx context.Context,
	params *executor.RunCommandRequest,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
) (err error) {
	// In docker, if network is not specified, we default to bridge
	if params.Network.Type == models.NetworkDefault {
		params.Network.Type = models.NetworkBridge
	}
	containerConfig.NetworkDisabled = params.Network.Disabled()
	switch params.Network.Type {
	case models.NetworkNone:
		hostConfig.NetworkMode = dockerNetworkNone

	case models.NetworkHost, models.NetworkFull:
		hostConfig.NetworkMode = dockerNetworkHost
		hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, dockerHostAddCommand)
		// In host mode, ports are directly accessible on the host network
		// No port bindings needed as container shares host's network namespace

	case models.NetworkBridge:
		hostConfig.NetworkMode = dockerNetworkBridge
		hostConfig.ExtraHosts = append(hostConfig.ExtraHosts, dockerHostAddCommand)

		// Add port bindings for bridge mode
		if len(params.Network.Ports) > 0 {
			portBindings := make(nat.PortMap)
			exposedPorts := make(nat.PortSet)

			for _, mapping := range params.Network.Ports {
				// In bridge mode, we use the Target port as the container port
				containerPort := nat.Port(fmt.Sprintf("%d/tcp", mapping.Target))

				// Use the host network from the mapping if specified, otherwise use all interfaces
				hostIP := "0.0.0.0"
				if mapping.HostNetwork != "" {
					hostIP = mapping.HostNetwork
				}

				hostBinding := nat.PortBinding{
					HostIP:   hostIP,
					HostPort: fmt.Sprintf("%d", mapping.Static),
				}

				portBindings[containerPort] = []nat.PortBinding{hostBinding}
				exposedPorts[containerPort] = struct{}{}
			}

			containerConfig.ExposedPorts = exposedPorts
			hostConfig.PortBindings = portBindings
		}

	case models.NetworkHTTP:
		var internalNetwork *network.Inspect
		var proxyAddr *net.TCPAddr
		internalNetwork, proxyAddr, err = e.createHTTPGateway(ctx, params.JobID, params.ExecutionID, params.Network)
		if err != nil {
			return
		}
		hostConfig.NetworkMode = container.NetworkMode(internalNetwork.Name)
		containerConfig.Env = append(containerConfig.Env,
			fmt.Sprintf("http_proxy=%s", proxyAddr.String()),
			fmt.Sprintf("https_proxy=%s", proxyAddr.String()),
		)

	default:
		err = fmt.Errorf("unsupported network type %q", params.Network.Type.String())
	}
	return
}

//nolint:funlen
func (e *Executor) createHTTPGateway(
	ctx context.Context,
	job string,
	executionID string,
	networkConfig *models.NetworkConfig,
) (*network.Inspect, *net.TCPAddr, error) {
	// Get the gateway image if we don't have it already
	err := e.client.PullImage(ctx, httpGatewayImage)
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "error pulling gateway image")
	}

	// Create an internal only bridge network to join our gateway and job container
	networkResp, err := e.client.NetworkCreate(ctx, e.dockerObjectName(executionID, job, "network"), network.CreateOptions{
		Driver:     "bridge",
		Scope:      "local",
		Internal:   true,
		Attachable: true,
		Labels:     e.containerLabels(executionID, job),
	})
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "error creating network")
	}

	// Get the subnet that Docker has picked for the newly created network
	internalNetwork, err := e.client.NetworkInspect(ctx, networkResp.ID, network.InspectOptions{})
	if err != nil || len(internalNetwork.IPAM.Config) < 1 {
		return nil, nil, pkgerrors.Wrap(err, "error getting network subnet")
	}
	subnet := internalNetwork.IPAM.Config[0].Subnet

	if len(networkConfig.DomainSet()) == 0 {
		return nil, nil,
			fmt.Errorf("invalid networking configuration, at least one domain is required when %s "+
				"networking is enabled", models.NetworkHTTP)
	}

	// Create the gateway container initially attached to the *host* network
	domainList, dErr := json.Marshal(networkConfig.DomainSet())
	clientList, cErr := json.Marshal([]string{subnet})
	if dErr != nil || cErr != nil {
		return nil, nil, pkgerrors.Wrap(errors.Join(dErr, cErr), "error preparing gateway config")
	}

	gatewayContainer, err := e.client.ContainerCreate(ctx, &container.Config{
		Image: httpGatewayImage,
		Env: []string{
			fmt.Sprintf("BACALHAU_HTTP_CLIENTS=%s", clientList),
			fmt.Sprintf("BACALHAU_HTTP_DOMAINS=%s", domainList),
			fmt.Sprintf("BACALHAU_JOB_ID=%s", job),
			fmt.Sprintf("BACALHAU_EXECUTION_ID=%s", executionID),
		},
		Healthcheck:     &container.HealthConfig{}, //TODO
		NetworkDisabled: false,
		Labels:          e.containerLabels(executionID, job),
	}, &container.HostConfig{
		NetworkMode: dockerNetworkBridge,
		CapAdd:      gatewayCapabilities,
		ExtraHosts:  []string{dockerHostAddCommand},
	}, nil, nil, e.dockerObjectName(executionID, job, "gateway"))
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "error creating gateway container")
	}

	// Attach the bridge network to the container
	err = e.client.NetworkConnect(ctx, internalNetwork.ID, gatewayContainer.ID, nil)
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "error attaching network to gateway")
	}

	// Start the container and wait for it to come up
	err = e.client.ContainerStart(ctx, gatewayContainer.ID, container.StartOptions{})
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to start network gateway container")
	}

	stdout, stderr, err := e.client.FollowLogs(ctx, gatewayContainer.ID)
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to get gateway container logs")
	}
	go logger.LogStream(log.Ctx(ctx).With().Str("Source", "stdout").Logger().WithContext(ctx), stdout)
	go logger.LogStream(log.Ctx(ctx).With().Str("Source", "stderr").Logger().WithContext(ctx), stderr)

	// Look up the IP address of the gateway container and attach it to the spec
	var containerDetails types.ContainerJSON
	for {
		containerDetails, err = e.client.ContainerInspect(ctx, gatewayContainer.ID)
		if err != nil {
			return nil, nil, pkgerrors.Wrap(err, "error getting gateway container details")
		}
		switch containerDetails.State.Health.Status {
		case types.NoHealthcheck:
			return nil, nil, errors.New("expecting gateway image to have healthcheck defined")
		case types.Unhealthy:
			return nil, nil, errors.New("gateway container failed to start")
		case types.Starting:
			time.Sleep(httpGatewayHealthcheckInterval)
			continue
		}

		break
	}

	networkAttachment, ok := containerDetails.NetworkSettings.Networks[internalNetwork.Name]
	if !ok || networkAttachment.IPAddress == "" {
		return nil, nil, fmt.Errorf("gateway does not appear to be attached to internal network")
	}
	proxyIP := net.ParseIP(networkAttachment.IPAddress)
	proxyAddr := net.TCPAddr{IP: proxyIP, Port: httpProxyPort}
	return &internalNetwork, &proxyAddr, err
}
