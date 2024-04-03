package nodefx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/authn/ask"
	"github.com/bacalhau-project/bacalhau/pkg/authn/challenge"
	"github.com/bacalhau-project/bacalhau/pkg/authz"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

type NodeConfig struct {
	NodeID          string
	Labels          map[string]string
	TransportConfig *nats_transport.NATSTransportConfig
	// ComputeConfig    *ComputeConfig
	RequesterConfig  *RequesterConfig
	EchoRouterConfig EchoRouterConfig
	ServerConfig     ServerConfig
	AuthConfig       *types.AuthConfig
}

type ServerConfig struct {
	Address            string
	Port               uint16
	AutoCertDomain     string
	AutoCertCache      string
	TLSCertificateFile string
	TLSKeyFile         string

	// These are TCP connection deadlines and not HTTP timeouts. They don't control the time it takes for our handlers
	// to complete. Deadlines operate on the connection, so our server will fail to return a result only after
	// the handlers try to access connection properties
	// ReadHeaderTimeout is the amount of time allowed to read request headers
	ReadHeaderTimeout time.Duration
	// ReadTimeout is the maximum duration for reading the entire request, including the body
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out writes of the response.
	// It doesn't cancel the context and doesn't stop handlers from running even after failing the request.
	// It is for added safety and should be a bit longer than the request handler timeout for better error handling.
	WriteTimeout time.Duration
}

type EchoRouterConfig struct {
	Headers                   map[string]string
	EchoMiddlewareConfig      EchoMiddlewareConfig
	TelemetryMiddlewareConfig TelemetryMiddlewareConfig
}

type BacalhauNode struct {
	Transport transport.TransportLayer
	Server    *Server
}

func (n *BacalhauNode) Interact() {
}

func NewNode(ctx context.Context, cfg *NodeConfig) error {
	// var bacalhauNode BacalhauNode
	var requester RequesterNode
	// var compute ComputeNode
	app := fx.New(
		fx.Supply(cfg),
		fx.Provide(NATSS),
		fx.Supply(cfg.NodeID),

		// this is essentially the API module, needs a few more endpoints
		fx.Provide(Authorizer),
		fx.Provide(NewEchoRouter),            // requires EchoRouterConfig
		fx.Invoke(NewAPIServer),              // requires echo and ServerConfig
		fx.Invoke(agent.InitAgentEndpoint),   // requires echo, nodeInfoProvider and DebugInforProviders
		fx.Invoke(shared.InitSharedEndpoint), // requires nodeID and nodeInforProvider

		fx.Provide(NodeInfoProvider),

		ProvideIf(Requester, cfg.RequesterConfig != nil),
		fx.Populate(&requester),
		// PopulateIf[RequesterNode](&requester, cfg.RequesterConfig != nil),
		// ProvideIf(Compute, cfg.ComputeConfig != nil),
		// PopulateIf[ComputeNode](&compute, cfg.ComputeConfig != nil),

		fx.Invoke(RegisterNodeInfoProviderDecorators),
		fx.Provide(AuthenticatorsProviders),
		fx.Invoke(func(router *echo.Echo, provider authn.Provider) {
			auth_endpoint.BindEndpoint(context.TODO(), router, provider)
		}),
		// fx.Populate(&bacalhauNode),
	)

	return app.Start(ctx)
}

func Authorizer(cfg *NodeConfig) (authz.Authorizer, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.AuthConfig.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := pkgconfig.GetClientPublicKey()
	if err != nil {
		return nil, err
	}
	return authz.NewPolicyAuthorizer(authzPolicy, signingKey, cfg.NodeID), nil
}

func ProvideIf(constructor func() fx.Option, condition bool) fx.Option {
	if condition {
		return constructor()
	}
	return fx.Options()
}

func PopulateIf[T any](instance *T, condition bool) fx.Option {
	if condition {
		fx.Populate(instance)
	}
	return fx.Options()
}

func NodeInfoProvider(cfg *NodeConfig) (*routing.NodeInfoProvider, error) {
	labelsProvider := models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: cfg.Labels},
		&node.RuntimeLabelsProvider{},
	)
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		NodeID:              cfg.NodeID,
		LabelsProvider:      labelsProvider,
		BacalhauVersion:     *version.Get(),
		DefaultNodeApproval: models.NodeApprovals.APPROVED,
	})
	return nodeInfoProvider, nil
}

func RegisterNodeInfoProviderDecorators(transport *nats_transport.NATSTransport, provider *routing.NodeInfoProvider) {
	provider.RegisterNodeInfoDecorator(transport.NodeInfoDecorator())
}

func AuthenticatorsProviders(cfg *NodeConfig) (authn.Provider, error) {
	var allErr error
	privKey, allErr := pkgconfig.GetClientPrivateKey()
	if allErr != nil {
		return nil, allErr
	}

	authns := make(map[string]authn.Authenticator, len(cfg.AuthConfig.Methods))
	for name, authnConfig := range cfg.AuthConfig.Methods {
		switch authnConfig.Type {
		case authn.MethodTypeChallenge:
			methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = challenge.NewAuthenticator(
				methodPolicy,
				challenge.NewStringMarshaller(cfg.NodeID),
				privKey,
				cfg.NodeID,
			)
		case authn.MethodTypeAsk:
			methodPolicy, err := policy.FromPath(authnConfig.PolicyPath)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = ask.NewAuthenticator(
				methodPolicy,
				privKey,
				cfg.NodeID,
			)
		default:
			allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
		}
	}

	return provider.NewMappedProvider(authns), allErr
}
