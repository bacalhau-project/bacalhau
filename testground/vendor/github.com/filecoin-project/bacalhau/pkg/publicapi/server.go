package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"

	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/transport"

	sync "github.com/lukemarsden/golang-mutex-tracer"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type APIServerConfig struct {
	// These are TCP connection deadlines and not HTTP timeouts. They don't control the time it takes for our handlers
	// to complete. Deadlines operate on the connection, so our server will fail to return a result only after
	// the handlers try to access connection properties
	ReadHeaderTimeout time.Duration // the amount of time allowed to read request headers
	ReadTimeout       time.Duration // the maximum duration for reading the entire request, including the body
	WriteTimeout      time.Duration // the maximum duration before timing out writes of the response

	// This represents maximum duration for handlers to complete, or else fail the request with 503 error code.
	RequestHandlerTimeout      time.Duration
	RequestHandlerTimeoutByURI map[string]time.Duration
}

var DefaultAPIServerConfig = &APIServerConfig{
	ReadHeaderTimeout:          10 * time.Second,
	ReadTimeout:                20 * time.Second,
	WriteTimeout:               20 * time.Second,
	RequestHandlerTimeout:      30 * time.Second,
	RequestHandlerTimeoutByURI: map[string]time.Duration{},
}

// APIServer configures a node's public REST API.
type APIServer struct {
	localdb          localdb.LocalDB
	transport        transport.Transport
	Requester        *requesternode.RequesterNode
	Publishers       publisher.PublisherProvider
	StorageProviders storage.StorageProvider
	Host             string
	Port             int
	Config           *APIServerConfig
}

func init() { //nolint:gochecknoinits
	sync.SetGlobalOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Enabled:   true,
		Id:        "<UNKNOWN>",
	})
}

// NewServer returns a new API server for a requester node.
func NewServer(
	ctx context.Context,
	host string,
	port int,
	localdb localdb.LocalDB,
	transport transport.Transport,
	requester *requesternode.RequesterNode,
	publishers publisher.PublisherProvider,
	storageProviders storage.StorageProvider,
) *APIServer {
	return NewServerWithConfig(
		ctx, host, port, localdb, transport, requester, publishers, storageProviders, DefaultAPIServerConfig)
}

func NewServerWithConfig(
	ctx context.Context,
	host string,
	port int,
	localdb localdb.LocalDB,
	transport transport.Transport,
	requester *requesternode.RequesterNode,
	publishers publisher.PublisherProvider,
	storageProviders storage.StorageProvider,
	config *APIServerConfig) *APIServer {
	a := &APIServer{
		localdb:          localdb,
		transport:        transport,
		Requester:        requester,
		Publishers:       publishers,
		StorageProviders: storageProviders,
		Host:             host,
		Port:             port,
		Config:           config,
	}
	return a
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *APIServer) GetURI() string {
	return fmt.Sprintf("http://%s:%d", apiServer.Host, apiServer.Port)
}

// ListenAndServe listens for and serves HTTP requests against the API server.
func (apiServer *APIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	hostID := apiServer.Requester.ID

	// TODO: #677 Significant issue, when client returns error to any of these commands, it still submits to server
	sm := http.NewServeMux()
	sm.Handle(apiServer.chainHandlers("/list", apiServer.list))
	sm.Handle(apiServer.chainHandlers("/states", apiServer.states))
	sm.Handle(apiServer.chainHandlers("/results", apiServer.results))
	sm.Handle(apiServer.chainHandlers("/events", apiServer.events))
	sm.Handle(apiServer.chainHandlers("/local_events", apiServer.localEvents))
	sm.Handle(apiServer.chainHandlers("/id", apiServer.id))
	sm.Handle(apiServer.chainHandlers("/peers", apiServer.peers))
	sm.Handle(apiServer.chainHandlers("/submit", apiServer.submit))
	sm.Handle(apiServer.chainHandlers("/version", apiServer.version))
	sm.Handle(apiServer.chainHandlers("/healthz", apiServer.healthz))
	sm.Handle(apiServer.chainHandlers("/logz", apiServer.logz))
	sm.Handle(apiServer.chainHandlers("/varz", apiServer.varz))
	sm.Handle(apiServer.chainHandlers("/livez", apiServer.livez))
	sm.Handle(apiServer.chainHandlers("/readyz", apiServer.readyz))
	sm.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Handler:           sm,
		Addr:              fmt.Sprintf("%s:%d", apiServer.Host, apiServer.Port),
		ReadHeaderTimeout: apiServer.Config.ReadHeaderTimeout,
		ReadTimeout:       apiServer.Config.ReadTimeout,
		WriteTimeout:      apiServer.Config.WriteTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return logger.ContextWithNodeIDLogger(context.Background(), apiServer.Requester.ID)
		},
	}

	log.Debug().Msgf(
		"API server listening for host %s on %s...", hostID, srv.Addr)

	// Cleanup resources when system is done:
	cm.RegisterCallback(func() error {
		return srv.Shutdown(ctx)
	})

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		log.Debug().Msgf(
			"API server closed for host %s on %s.", hostID, srv.Addr)
		return nil // expected error if the server is shut down
	}

	return err
}

func verifySubmitRequest(req *submitRequest) error {
	if req.Data.ClientID == "" {
		return errors.New("job deal must contain a client ID")
	}
	if req.ClientSignature == "" {
		return errors.New("client's signature is required")
	}
	if req.ClientPublicKey == "" {
		return errors.New("client's public key is required")
	}

	// Check that the client's public key matches the client ID:
	ok, err := system.PublicKeyMatchesID(req.ClientPublicKey, req.Data.ClientID)
	if err != nil {
		return fmt.Errorf("error verifying client ID: %w", err)
	}
	if !ok {
		return errors.New("client's public key does not match client ID")
	}

	// Check that the signature is valid:
	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return fmt.Errorf("error marshaling job data: %w", err)
	}

	err = system.Verify(jsonData, req.ClientSignature, req.ClientPublicKey)
	if err != nil {
		return fmt.Errorf("client's signature is invalid: %w", err)
	}

	return nil
}

func (apiServer *APIServer) chainHandlers(uri string, handlerFunc http.HandlerFunc) (string, http.Handler) {
	// otel handler
	handler := otelhttp.NewHandler(handlerFunc, fmt.Sprintf("pkg/publicapi%s", uri))

	// throttling handler
	handler = tollbooth.LimitHandler(
		tollbooth.NewLimiter(
			1000, //nolint:gomnd
			&limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}),
		handler)

	// timeout handler. Find timeout for this endpoint, or use the fallback value
	handlerTimeout, ok := apiServer.Config.RequestHandlerTimeoutByURI[uri]
	if !ok {
		if apiServer.Config.RequestHandlerTimeout != 0 {
			handlerTimeout = apiServer.Config.RequestHandlerTimeout
		} else {
			// if no fallback timeout is defined, then use the default value
			handlerTimeout = DefaultAPIServerConfig.RequestHandlerTimeout
		}
	}
	handler = http.TimeoutHandler(handler, handlerTimeout, "Server Timeout!")

	// logging handler. Should be last in the chain.
	handler = handlerwrapper.NewHTTPHandlerWrapper(apiServer.Requester.ID, handler, handlerwrapper.NewJSONLogHandler())
	return uri, handler
}
