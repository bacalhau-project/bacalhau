package transport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
)

const NATSServerDefaultTLSTimeout = 10

// reservedChars are the characters that are not allowed in node IDs as nodes
// subscribe to subjects with their node IDs, and these are wildcards
// in NATS subjects that could cause a node to subscribe to unintended subjects.
const reservedChars = ".*>"

type NATSTransportConfig struct {
	NodeID            string
	Host              string
	Port              int
	AdvertisedAddress string
	Orchestrators     []string
	IsRequesterNode   bool

	// StoreDir is the directory where the NATS server will store its data
	StoreDir string

	// AuthSecret is a secret string that clients must use to connect. NATS servers
	// must supply this config, while clients can also supply it as the user part
	// of their Orchestrator URL.
	AuthSecret string

	// Cluster config for requester nodes to connect with each other
	ClusterName              string
	ClusterPort              int
	ClusterAdvertisedAddress string
	ClusterPeers             []string

	// TLS
	ServerTLSCert    string
	ServerTLSKey     string
	ServerTLSTimeout int

	// Used by the Nats Client when node acts as orchestrator
	ServerTLSCACert string

	// Used by the Nats Client when node acts as compute
	ClientTLSCACert string

	// Used to configure Orchestrator (actually the NATS server) to run behind
	// a reverse proxy
	ServerSupportReverseProxy bool

	// Used to configure compute node nats client to require TLS connection
	ComputeClientRequireTLS bool
}

func (c *NATSTransportConfig) Validate() error {
	mErr := errors.Join(
		validate.NotBlank(c.NodeID, "missing node ID"),
		validate.NoSpaces(c.NodeID, "node ID cannot contain spaces"),
		validate.NoNullChars(c.NodeID, "node ID cannot contain null characters"),
		validate.ContainsNoneOf(c.NodeID, reservedChars,
			"node ID cannot contain any of the following characters: %s", reservedChars),
	)

	if c.IsRequesterNode {
		mErr = errors.Join(mErr, validate.IsGreaterThanZero(c.Port, "port %d must be greater than zero", c.Port))

		// if cluster config is set, validate it
		if c.ClusterName != "" || c.ClusterPort != 0 || c.ClusterAdvertisedAddress != "" || len(c.ClusterPeers) > 0 {
			mErr = errors.Join(mErr,
				validate.IsGreaterThanZero(c.ClusterPort, "cluster port %d must be greater than zero", c.ClusterPort))
		}
	} else {
		mErr = errors.Join(mErr, validate.IsNotEmpty(c.Orchestrators, "missing orchestrators"))
	}

	serverCertProvided := c.ServerTLSCert != ""
	serverKeyProvided := c.ServerTLSKey != ""

	if serverCertProvided != serverKeyProvided {
		mErr = errors.Join(
			mErr,
			fmt.Errorf("both ServerTLSCert and ServerTLSKey must be set together"),
		)
	}

	if serverCertProvided && serverKeyProvided && c.ServerTLSTimeout < 0 {
		mErr = errors.Join(
			mErr,
			fmt.Errorf("NATS ServerTLSTimeout must be a positive number, got: %d", c.ServerTLSTimeout),
		)
	}

	if mErr != nil {
		return nats_helper.NewConfigurationError("invalid transport config:\n%s", mErr)
	}
	return nil
}

type NATSTransport struct {
	Config     *NATSTransportConfig
	nodeID     string
	natsServer *nats_helper.ServerManager
}

//nolint:funlen
func NewNATSTransport(ctx context.Context,
	config *NATSTransportConfig) (*NATSTransport, error) {
	log.Debug().Msgf("Creating NATS transport with config: %+v", config)
	if err := config.Validate(); err != nil {
		return nil, bacerrors.Wrap(err, "invalid cluster config").WithCode(bacerrors.ValidationError)
	}

	var sm *nats_helper.ServerManager
	if config.IsRequesterNode {
		var err error

		// create nats server with servers acting as its cluster peers
		serverOpts := &server.Options{
			ServerName:             config.NodeID,
			Host:                   config.Host,
			Port:                   config.Port,
			ClientAdvertise:        config.AdvertisedAddress,
			Authorization:          config.AuthSecret,
			Debug:                  true, // will only be used if log level is debug
			JetStream:              true,
			DisableJetStreamBanner: true,
			StoreDir:               config.StoreDir,
			NoSigs:                 true, // disable terminating the server on SIGINT/SIGTERM
		}

		if config.ServerTLSCert != "" {
			serverTLSTimeout := NATSServerDefaultTLSTimeout
			if config.ServerTLSTimeout > 0 {
				serverTLSTimeout = config.ServerTLSTimeout
			}

			serverTLSConfigOpts := &server.TLSConfigOpts{
				CertFile: config.ServerTLSCert,
				KeyFile:  config.ServerTLSKey,
				Timeout:  float64(serverTLSTimeout),
			}

			serverTLSConfig, err := server.GenTLSConfig(serverTLSConfigOpts)
			if err != nil {
				log.Error().Msgf("failed to configure NATS server TLS: %v", err)
				return nil, err
			}

			serverOpts.TLSConfig = serverTLSConfig
		}

		if config.ServerSupportReverseProxy {
			// If the ServerSupportReverseProxy is enabled, we need to set
			// serverOpts.TLSConfig to an empty config, if it is null.
			// Reason for that , unfortunately that's the only eay NATS server will
			// work behind a reverse proxy, that's how NATS documentation recommends doing it.
			// See: https://docs.nats.io/running-a-nats-service/configuration/securing_nats/tls#tls-terminating-reverse-proxies
			serverOpts.AllowNonTLS = true

			// We need to make sure not to override TLS configuration if it was set. Maybe the operator want TLS
			// between reverse proxy and NATS server, up to them.
			if serverOpts.TLSConfig == nil {
				serverOpts.TLSConfig, _ = server.GenTLSConfig(&server.TLSConfigOpts{})
			}
		}

		// Only set cluster options if cluster peers are provided. Jetstream doesn't
		// like the setting to be present with no values, or with values that are
		// a local address (e.g. it can't RAFT to itself).
		routes, err := nats_helper.RoutesFromSlice(config.ClusterPeers, false)
		if err != nil {
			return nil, err
		}

		if len(config.ClusterPeers) > 0 {
			serverOpts.Routes = routes

			serverOpts.Cluster = server.ClusterOpts{
				Name:      config.ClusterName,
				Port:      config.ClusterPort,
				Advertise: config.ClusterAdvertisedAddress,
			}
		}

		log.Debug().Msgf("Creating NATS server with options: %+v", serverOpts)
		sm, err = nats_helper.NewServerManager(ctx, nats_helper.ServerManagerParams{
			Options: serverOpts,
		})
		if err != nil {
			return nil, err
		}

		if config.ServerSupportReverseProxy {
			// Server.ClientURL() (in core NATS code), will check if TLSConfig of the server
			// is not null, and changes the URL Scheme from "nats" to "tls". When running
			// the server with ServerSupportReverseProxy setting, almost all the time
			// the NATS server will not be supporting TLS. This will make the orchestrator NATS client
			// fail, since it was given the "tls://" NATS server URL to connect to, but the
			// server does not support TLS. It is unfortunate that the ClientURL method does that.
			// So here, we are checking, if NATS server was not started with a cert and key, and at the
			// same time it was started with ServerSupportReverseProxy set to true, then we will change
			// URL the scheme back to "nats://" from "tls://".

			clientURL := sm.Server.ClientURL()
			if strings.HasPrefix(clientURL, "tls://") && config.ServerTLSCert == "" {
				clientURL = strings.Replace(clientURL, "tls://", "nats://", 1)
			}
			config.Orchestrators = append(config.Orchestrators, clientURL)
		} else {
			config.Orchestrators = append(config.Orchestrators, sm.Server.ClientURL())
		}
	}

	// create transport
	return &NATSTransport{
		nodeID:     config.NodeID,
		natsServer: sm,
		Config:     config,
	}, nil

}

// CreateClient creates a new NATS client.
func (t *NATSTransport) CreateClient(ctx context.Context) (*nats.Conn, error) {
	clientManager, err := CreateClient(ctx, t.Config)
	if err != nil {
		return nil, err
	}
	return clientManager.Client, nil
}

func CreateClient(ctx context.Context, config *NATSTransportConfig) (*nats_helper.ClientManager, error) {
	// create nats client
	log.Debug().Msgf("Creating NATS client with servers: %s", strings.Join(config.Orchestrators, ","))
	clientOptions := []nats.Option{
		nats.Name(config.NodeID),
		nats.MaxReconnects(-1),
	}

	// When Compute Node requires TLS, enforce it
	if config.ComputeClientRequireTLS {
		clientOptions = append(clientOptions, nats.TLSHandshakeFirst())
	}

	// We need to do this logic since the Nats Transport Layer does not differentiate
	// between orchestrator mode and compute mode
	if config.ServerTLSCert == "" && config.ClientTLSCACert != "" {
		// this client is for a compute node
		clientOptions = append(clientOptions, nats.RootCAs(config.ClientTLSCACert))
	} else if config.ServerTLSCert != "" && config.ServerTLSCACert != "" {
		// this client is for a orchestrator node
		clientOptions = append(clientOptions, nats.RootCAs(config.ServerTLSCACert))
	}

	if config.AuthSecret != "" {
		clientOptions = append(clientOptions, nats.Token(config.AuthSecret))
	}
	return nats_helper.NewClientManager(ctx,
		strings.Join(config.Orchestrators, ","),
		clientOptions...,
	)
}

// DebugInfoProviders returns the debug info of the NATS transport layer
func (t *NATSTransport) DebugInfoProviders() []models.DebugInfoProvider {
	var debugInfoProviders []models.DebugInfoProvider
	if t.natsServer != nil {
		debugInfoProviders = append(debugInfoProviders, t.natsServer)
	}
	return debugInfoProviders
}

// Close closes the transport layer.
func (t *NATSTransport) Close(ctx context.Context) error {
	if t.natsServer != nil {
		log.Ctx(ctx).Debug().Msgf("Shutting down server %s", t.natsServer.Server.Name())
		t.natsServer.Stop()
	}
	return nil
}
