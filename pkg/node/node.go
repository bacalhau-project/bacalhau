package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	baccrypto "github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/metrics"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const (
	maxPortNumber = 65535
	minPortNumber = 0
)

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}

type MetadataStore interface {
	ReadLastUpdateCheck() (time.Time, error)
	WriteLastUpdateCheck(time.Time) error
	InstanceID() string
}

type NodeConfig struct {
	NodeID                 string
	CleanupManager         *system.CleanupManager
	BacalhauConfig         types.Bacalhau
	SystemConfig           SystemConfig
	DependencyInjector     NodeDependencyInjector
	FailureInjectionConfig models.FailureInjectionConfig
}

func (c *NodeConfig) Validate() error {
	// TODO: add more validations
	var mErr error
	mErr = errors.Join(mErr, validate.NotBlank(c.NodeID, "node id is required"))
	return mErr
}

// NodeDependencyInjector Lazy node dependency injector that generate instances of different
// components on demand and based on the configuration provided.
type NodeDependencyInjector struct {
	StorageProvidersFactory StorageProvidersFactory
	ExecutorsFactory        ExecutorsFactory
	PublishersFactory       PublishersFactory
	AuthenticatorsFactory   AuthenticatorsFactory
}

func NewStandardNodeDependencyInjector(cfg types.Bacalhau, userKey *baccrypto.UserKey) NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(cfg),
		ExecutorsFactory:        NewStandardExecutorsFactory(cfg.Engines),
		PublishersFactory:       NewStandardPublishersFactory(cfg),
		AuthenticatorsFactory:   NewStandardAuthenticatorsFactory(userKey),
	}
}

type Node struct {
	// Visible for testing
	ID             string
	APIServer      *publicapi.Server
	ComputeNode    *Compute
	RequesterNode  *Requester
	CleanupManager *system.CleanupManager
}

func (n *Node) Start(ctx context.Context) error {
	return n.APIServer.ListenAndServe(ctx)
}

//nolint:funlen,gocyclo // Should be simplified when moving to FX
func NewNode(
	ctx context.Context,
	cfg NodeConfig,
	metadataStore MetadataStore,
) (*Node, error) {
	var err error
	if err = cfg.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	// apply default values to the system config
	cfg.SystemConfig.applyDefaults()
	log.Ctx(ctx).Debug().Msgf("Starting node %s with config: %+v", cfg.NodeID, cfg.BacalhauConfig)

	userKeyPath, err := cfg.BacalhauConfig.UserKeyPath()
	if err != nil {
		return nil, err
	}
	userKey, err := baccrypto.LoadUserKey(userKeyPath)
	if err != nil {
		return nil, err
	}

	cfg.DependencyInjector =
		mergeDependencyInjectors(cfg.DependencyInjector, NewStandardNodeDependencyInjector(cfg.BacalhauConfig, userKey))

	apiServer, err := createAPIServer(cfg, userKey)
	if err != nil {
		return nil, err
	}

	transportLayer, err := createTransport(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var debugInfoProviders []models.DebugInfoProvider
	debugInfoProviders = append(debugInfoProviders, transportLayer.DebugInfoProviders()...)

	var requesterNode *Requester
	var computeNode *Compute

	// Create a node info provider
	nodeInfoProvider := models.NewBaseNodeInfoProvider(models.BaseNodeInfoProviderParams{
		NodeID:             cfg.NodeID,
		BacalhauVersion:    *version.Get(),
		SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2, models.ProtocolNCLV1},
	})
	nodeInfoProvider.RegisterLabelProvider(&ConfigLabelsProvider{staticLabels: cfg.BacalhauConfig.Labels})
	nodeInfoProvider.RegisterLabelProvider(&RuntimeLabelsProvider{})

	// setup requester node
	if cfg.BacalhauConfig.Orchestrator.Enabled {
		requesterNode, err = NewRequesterNode(
			ctx,
			cfg,
			apiServer,
			transportLayer,
			metadataStore,
			nodeInfoProvider,
		)
		if err != nil {
			return nil, err
		}

		debugInfoProviders = append(debugInfoProviders, requesterNode.debugInfoProviders...)
	}

	if cfg.BacalhauConfig.Compute.Enabled {
		// setup compute node
		computeNode, err = NewComputeNode(
			ctx,
			cfg,
			apiServer,
			transportLayer,
			nodeInfoProvider,
		)
		if err != nil {
			return nil, err
		}

		debugInfoProviders = append(debugInfoProviders, computeNode.debugInfoProviders...)
	}

	shared.NewEndpoint(shared.EndpointParams{
		Router:           apiServer.Router,
		NodeID:           cfg.NodeID,
		NodeInfoProvider: nodeInfoProvider,
	})

	agent.NewEndpoint(agent.EndpointParams{
		Router:             apiServer.Router,
		NodeInfoProvider:   nodeInfoProvider,
		DebugInfoProviders: debugInfoProviders,
		BacalhauConfig:     cfg.BacalhauConfig,
	})

	// Start periodic software update checks.
	version.RunUpdateChecker(
		ctx,
		cfg.BacalhauConfig,
		metadataStore,
		func(ctx context.Context) (*models.BuildVersionInfo, error) { return nil, nil },
		version.LogUpdateResponse,
	)

	// Cleanup libp2p resources in the desired order
	cfg.CleanupManager.RegisterCallbackWithContext(func(ctx context.Context) error {
		if computeNode != nil {
			computeNode.Cleanup(ctx)
		}
		if requesterNode != nil {
			requesterNode.cleanup(ctx)
		}

		var err error
		if transportLayer != nil {
			err = errors.Join(err, transportLayer.Close(ctx))
		}

		if apiServer != nil {
			err = errors.Join(err, apiServer.Shutdown(ctx))
		}
		cancel()
		return err
	})

	metrics.NodeInfo.Add(ctx, 1,
		attribute.String("node_id", cfg.NodeID),
		attribute.Bool("node_is_compute", cfg.BacalhauConfig.Compute.Enabled),
		attribute.Bool("node_is_orchestrator", cfg.BacalhauConfig.Orchestrator.Enabled),
	)
	node := &Node{
		ID:             cfg.NodeID,
		CleanupManager: cfg.CleanupManager,
		APIServer:      apiServer,
		ComputeNode:    computeNode,
		RequesterNode:  requesterNode,
	}

	return node, nil
}

func createAPIServer(cfg NodeConfig, userKey *baccrypto.UserKey) (*publicapi.Server, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.BacalhauConfig.API.Auth.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	givenPortNumber := cfg.BacalhauConfig.API.Port
	if givenPortNumber < minPortNumber {
		return nil, fmt.Errorf("invalid negative port number: %d", cfg.BacalhauConfig.API.Port)
	}
	if givenPortNumber > maxPortNumber {
		return nil, fmt.Errorf("port number %d exceeds maximum allowed value (65535)", cfg.BacalhauConfig.API.Port)
	}

	safePortNumber := uint16(givenPortNumber)

	serverVersion := version.Get()
	// public http api server
	serverParams := publicapi.ServerParams{
		Router:     echo.New(),
		Address:    cfg.BacalhauConfig.API.Host,
		Port:       safePortNumber,
		HostID:     cfg.NodeID,
		Config:     publicapi.DefaultConfig(), // using default as we don't expose this config to the user
		Authorizer: authz.NewPolicyAuthorizer(authzPolicy, userKey.PublicKey(), cfg.NodeID),
		Headers: map[string]string{
			apimodels.HTTPHeaderBacalhauGitVersion: serverVersion.GitVersion,
			apimodels.HTTPHeaderBacalhauGitCommit:  serverVersion.GitCommit,
			apimodels.HTTPHeaderBacalhauBuildDate:  serverVersion.BuildDate.UTC().String(),
			apimodels.HTTPHeaderBacalhauBuildOS:    serverVersion.GOOS,
			apimodels.HTTPHeaderBacalhauArch:       serverVersion.GOARCH,
		},
	}

	// Only allow autocert for requester nodes
	if cfg.BacalhauConfig.Orchestrator.Enabled {
		serverParams.AutoCertDomain = cfg.BacalhauConfig.API.TLS.AutoCert
		serverParams.AutoCertCache = cfg.BacalhauConfig.API.TLS.AutoCertCachePath
		// If there are configuration values for autocert we should return and let autocert
		// do what it does later on in the setup.
		if serverParams.AutoCertDomain == "" {
			cert, key, err := getTLSCertificate(cfg.BacalhauConfig)
			if err != nil {
				return nil, err
			}
			serverParams.TLSCertificateFile = cert
			serverParams.TLSKeyFile = key
		}
	}

	apiServer, err := publicapi.NewAPIServer(serverParams)
	if err != nil {
		return nil, err
	}
	return apiServer, nil
}

func createTransport(ctx context.Context, cfg NodeConfig) (*nats_transport.NATSTransport, error) {
	storeDir, err := cfg.BacalhauConfig.NetworkTransportDir()
	if err != nil {
		return nil, err
	}

	// TODO: revisit how we setup the transport layer for compute only, orchestrator only and hybrid nodes
	config := &nats_transport.NATSTransportConfig{
		NodeID:                    cfg.NodeID,
		Host:                      cfg.BacalhauConfig.Orchestrator.Host,
		Port:                      cfg.BacalhauConfig.Orchestrator.Port,
		AdvertisedAddress:         cfg.BacalhauConfig.Orchestrator.Advertise,
		AuthSecret:                cfg.BacalhauConfig.Orchestrator.Auth.Token,
		Orchestrators:             cfg.BacalhauConfig.Compute.Orchestrators,
		StoreDir:                  storeDir,
		ClusterName:               cfg.BacalhauConfig.Orchestrator.Cluster.Name,
		ClusterPort:               cfg.BacalhauConfig.Orchestrator.Cluster.Port,
		ClusterPeers:              cfg.BacalhauConfig.Orchestrator.Cluster.Peers,
		ClusterAdvertisedAddress:  cfg.BacalhauConfig.Orchestrator.Cluster.Advertise,
		IsRequesterNode:           cfg.BacalhauConfig.Orchestrator.Enabled,
		ServerTLSCACert:           cfg.BacalhauConfig.Orchestrator.TLS.CACert,
		ServerTLSCert:             cfg.BacalhauConfig.Orchestrator.TLS.ServerCert,
		ServerTLSKey:              cfg.BacalhauConfig.Orchestrator.TLS.ServerKey,
		ServerTLSTimeout:          cfg.BacalhauConfig.Orchestrator.TLS.ServerTimeout,
		ServerSupportReverseProxy: cfg.BacalhauConfig.Orchestrator.SupportReverseProxy,
		ClientTLSCACert:           cfg.BacalhauConfig.Compute.TLS.CACert,
		ComputeClientRequireTLS:   cfg.BacalhauConfig.Compute.TLS.RequireTLS,
	}

	if cfg.BacalhauConfig.Compute.Enabled && !cfg.BacalhauConfig.Orchestrator.Enabled {
		config.AuthSecret = cfg.BacalhauConfig.Compute.Auth.Token
	}

	transportLayer, err := nats_transport.NewNATSTransport(ctx, config)
	if err != nil {
		return nil, bacerrors.Wrap(err, "failed to create transport layer")
	}
	return transportLayer, nil
}

// IsRequesterNode returns true if the node is a requester node
func (n *Node) IsRequesterNode() bool {
	return n.RequesterNode != nil
}

// IsComputeNode returns true if the node is a compute node
func (n *Node) IsComputeNode() bool {
	return n.ComputeNode != nil
}

func mergeDependencyInjectors(injector NodeDependencyInjector, defaultInjector NodeDependencyInjector) NodeDependencyInjector {
	if injector.StorageProvidersFactory == nil {
		injector.StorageProvidersFactory = defaultInjector.StorageProvidersFactory
	}
	if injector.ExecutorsFactory == nil {
		injector.ExecutorsFactory = defaultInjector.ExecutorsFactory
	}
	if injector.PublishersFactory == nil {
		injector.PublishersFactory = defaultInjector.PublishersFactory
	}
	if injector.AuthenticatorsFactory == nil {
		injector.AuthenticatorsFactory = defaultInjector.AuthenticatorsFactory
	}
	return injector
}
