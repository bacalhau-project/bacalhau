package publicapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/c2h5oh/datasize"
	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/filecoin-project/bacalhau/docs"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi/handlerwrapper"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var DefaultAPIServerConfig = APIServerConfig{
	ReadHeaderTimeout:          10 * time.Second,
	ReadTimeout:                20 * time.Second,
	WriteTimeout:               20 * time.Second,
	RequestHandlerTimeout:      30 * time.Second,
	RequestHandlerTimeoutByURI: map[string]time.Duration{},
	MaxBytesToReadInBody:       10 * datasize.MB,
}

type HandlerConfig struct {
	URI                   string
	Handler               http.Handler
	RequestHandlerTimeout time.Duration
	Raw                   bool // don't wrap the handler with middleware
}

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

	// MaxBytesToReadInBody is used by safeHandlerFuncWrapper as the max size of body
	MaxBytesToReadInBody datasize.ByteSize
}

type APIServerParams struct {
	Address          string
	Port             int
	Host             host.Host
	NodeInfoProvider model.NodeInfoProvider
	Config           APIServerConfig
}

// APIServer configures a node's public REST API.
type APIServer struct {
	Address          string
	Port             int
	host             host.Host
	nodeInfoProvider model.NodeInfoProvider
	config           APIServerConfig
	handlers         map[string]http.Handler
	handlersMu       sync.Mutex
	started          bool
}

func NewAPIServer(params APIServerParams) (*APIServer, error) {
	server := &APIServer{
		Address:          params.Address,
		Port:             params.Port,
		host:             params.Host,
		nodeInfoProvider: params.NodeInfoProvider,
		config:           params.Config,
		handlers:         make(map[string]http.Handler),
	}

	server.handlersMu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "APIServer.handlersMu",
	})

	// dynamically write the git tag to the Swagger docs
	docs.SwaggerInfo.Version = version.Get().GitVersion

	// Register default handlers
	handlerConfigs := []HandlerConfig{
		{URI: "/id", Handler: http.HandlerFunc(server.id)},
		{URI: "/peers", Handler: http.HandlerFunc(server.peers)},
		{URI: "/node_info", Handler: http.HandlerFunc(server.nodeInfo)},
		{URI: "/version", Handler: http.HandlerFunc(server.version)},
		{URI: "/healthz", Handler: http.HandlerFunc(server.healthz)},
		{URI: "/logz", Handler: http.HandlerFunc(server.logz)},
		{URI: "/varz", Handler: http.HandlerFunc(server.varz)},
		{URI: "/livez", Handler: http.HandlerFunc(server.livez)},
		{URI: "/readyz", Handler: http.HandlerFunc(server.readyz)},
		{URI: "/swagger/", Handler: httpSwagger.WrapHandler, Raw: true},
	}
	err := server.RegisterHandlers(handlerConfigs...)
	if err != nil {
		return nil, err
	}

	return server, nil
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *APIServer) GetURI() string {
	return fmt.Sprintf("http://%s:%d", apiServer.Address, apiServer.Port)
}

//	@title			Bacalhau API
//	@description	This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/filecoin-project/bacalhau.
//	@contact.name	Bacalhau Team
//	@contact.url	https://github.com/filecoin-project/bacalhau
//	@contact.email	team@bacalhau.org
//	@license.name	Apache 2.0
//	@license.url	https://github.com/filecoin-project/bacalhau/blob/main/LICENSE
//	@host			bootstrap.production.bacalhau.org:1234
//	@BasePath		/
//	@schemes		http
//
// ListenAndServe listens for and serves HTTP requests against the API server.
//
//nolint:lll
func (apiServer *APIServer) ListenAndServe(ctx context.Context, cm *system.CleanupManager) error {
	apiServer.handlersMu.Lock()
	if apiServer.started {
		apiServer.handlersMu.Unlock()
		return fmt.Errorf("api server already started")
	}

	// TODO: #677 Significant issue, when client returns error to any of these commands, it still submits to server
	sm := http.NewServeMux()
	for uri, handler := range apiServer.handlers {
		sm.Handle(uri, handler)
	}
	apiServer.handlersMu.Unlock()

	srv := http.Server{
		Handler:           sm,
		ReadHeaderTimeout: apiServer.config.ReadHeaderTimeout,
		ReadTimeout:       apiServer.config.ReadTimeout,
		WriteTimeout:      apiServer.config.WriteTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return logger.ContextWithNodeIDLogger(context.Background(), apiServer.host.ID().String())
		},
	}

	addr := fmt.Sprintf("%s:%d", apiServer.Address, apiServer.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if apiServer.Port == 0 {
		switch addr := listener.Addr().(type) {
		case *net.TCPAddr:
			apiServer.Port = addr.Port
		default:
			return fmt.Errorf("unknown address %v", addr)
		}
	}

	log.Ctx(ctx).Debug().Msgf(
		"API server listening for host %s on %s...", apiServer.Address, listener.Addr().String())

	// Cleanup resources when system is done:
	cm.RegisterCallbackWithContext(srv.Shutdown)

	err = srv.Serve(listener)
	if err == http.ErrServerClosed {
		log.Ctx(ctx).Debug().Msgf(
			"API server closed for host %s on %s.", apiServer.Address, srv.Addr)
		return nil // expected error if the server is shut down
	}

	return err
}

func (apiServer *APIServer) RegisterHandlers(config ...HandlerConfig) error {
	apiServer.handlersMu.Lock()
	defer apiServer.handlersMu.Unlock()
	for _, c := range config {
		if err := apiServer.registerHandler(c); err != nil {
			return err
		}
	}
	return nil
}

func (apiServer *APIServer) registerHandler(config HandlerConfig) error {
	if _, ok := apiServer.handlers[config.URI]; ok {
		return fmt.Errorf("handler already registered for %s", config.URI)
	}
	if apiServer.started {
		return fmt.Errorf("cannot register new handlers after starting the api server")
	}

	handler := config.Handler
	if !config.Raw {
		// otel handler
		handler = otelhttp.NewHandler(config.Handler, config.URI,
			otelhttp.WithPublicEndpoint(),
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, operation)
			}),
		)

		// throttling handler
		handler = tollbooth.LimitHandler(
			tollbooth.NewLimiter(
				1000, //nolint:gomnd
				&limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour}),
			handler)

		// timeout handler. Find timeout for this endpoint, or use the fallback value
		handlerTimeout := config.RequestHandlerTimeout
		if handlerTimeout == 0 {
			handlerTimeout = apiServer.config.RequestHandlerTimeoutByURI[config.URI]
		}
		if handlerTimeout == 0 {
			handlerTimeout = apiServer.config.RequestHandlerTimeout
		}
		if handlerTimeout == 0 {
			handlerTimeout = DefaultAPIServerConfig.RequestHandlerTimeout
		}
		handler = http.TimeoutHandler(handler, handlerTimeout, "Server Timeout!")
		handler = http.MaxBytesHandler(handler, int64(apiServer.config.MaxBytesToReadInBody))

		// logging handler. Should be last in the chain.
		handler = handlerwrapper.NewHTTPHandlerWrapper(apiServer.host.ID().String(), handler, handlerwrapper.NewJSONLogHandler())
	}
	apiServer.handlers[config.URI] = handler
	return nil
}
