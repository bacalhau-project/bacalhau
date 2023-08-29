package publicapi

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

const TimeoutMessage = "Server Timeout!"

type ServerParams struct {
	Router  *echo.Echo
	Address string
	Port    uint16
	HostID  string
	Config  Config
}

// Server configures a node's public REST API.
type Server struct {
	Router  *echo.Echo
	Address string
	Port    uint16

	httpServer http.Server
	config     Config
}

func NewAPIServer(params ServerParams) (*Server, error) {
	server := &Server{
		Router:  params.Router,
		Address: params.Address,
		Port:    params.Port,
		config:  params.Config,
	}

	// migrate old endpoints to new versioned ones
	migrations := map[string]string{
		"/peers":                      "/api/v1/peers",
		"/node_info":                  "/api/v1/node_info",
		"/version":                    "/api/v1/version",
		"/healthz":                    "/api/v1/healthz",
		"/id":                         "/api/v1/id",
		"/livez":                      "/api/v1/livez",
		"/requester/list":             "/api/v1/requester/list",
		"/requester/nodes":            "/api/v1/requester/nodes",
		"/requester/states":           "/api/v1/requester/states",
		"/requester/results":          "/api/v1/requester/results",
		"/requester/events":           "/api/v1/requester/events",
		"/requester/submit":           "/api/v1/requester/submit",
		"/requester/cancel":           "/api/v1/requester/cancel",
		"/requester/debug":            "/api/v1/requester/debug",
		"/requester/logs":             "/api/v1/requester/logs",
		"/requester/websocket/events": "/api/v1/requester/websocket/events",
	}

	// TODO: #830 Same as #829 in pkg/eventhandler/chained_handlers.go
	logErrorStatusesOnly := system.GetEnvironment() != system.EnvironmentTest &&
		system.GetEnvironment() != system.EnvironmentDev

	// base middleware stack
	server.Router.Pre(
		echomiddelware.TimeoutWithConfig(echomiddelware.TimeoutConfig{
			Timeout:      params.Config.RequestHandlerTimeout,
			ErrorMessage: TimeoutMessage,
			Skipper:      middleware.PathMatchSkipper(params.Config.SkippedTimeoutPaths),
		}),
		echomiddelware.RateLimiter(echomiddelware.NewRateLimiterMemoryStore(rate.Limit(params.Config.ThrottleLimit))),
		echomiddelware.RequestID(),
		middleware.RequestLogger(logErrorStatusesOnly),
		middleware.Otel(),
		echomiddelware.Rewrite(migrations), // after logger and otel to track old paths
		echomiddelware.BodyLimit(server.config.MaxBytesToReadInBody),
	)

	server.httpServer = http.Server{
		Handler:           server.Router,
		ReadHeaderTimeout: server.config.ReadHeaderTimeout,
		ReadTimeout:       server.config.ReadTimeout,
		WriteTimeout:      server.config.WriteTimeout,
		BaseContext: func(l net.Listener) context.Context {
			return logger.ContextWithNodeIDLogger(context.Background(), params.HostID)
		},
	}

	return server, nil
}

// GetURI returns the HTTP URI that the server is listening on.
func (apiServer *Server) GetURI() *url.URL {
	interpolated := fmt.Sprintf("%s://%s:%d", apiServer.config.Protocol, apiServer.Address, apiServer.Port)
	url, err := url.Parse(interpolated)
	if err != nil {
		panic(fmt.Errorf("callback url must parse: %s", interpolated))
	}
	return url
}

//	@title			Bacalhau API
//	@description	This page is the reference of the Bacalhau REST API. Project docs are available at https://docs.bacalhau.org/. Find more information about Bacalhau at https://github.com/bacalhau-project/bacalhau.
//	@contact.name	Bacalhau Team
//	@contact.url	https://github.com/bacalhau-project/bacalhau
//	@contact.email	team@bacalhau.org
//	@license.name	Apache 2.0
//	@license.url	https://github.com/bacalhau-project/bacalhau/blob/main/LICENSE
//	@host			bootstrap.production.bacalhau.org:1234
//	@BasePath		/
//	@schemes		http
//
// ListenAndServe listens for and serves HTTP requests against the API server.
//
//nolint:lll
func (apiServer *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", apiServer.Address, apiServer.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if apiServer.Port == 0 {
		switch addr := listener.Addr().(type) {
		case *net.TCPAddr:
			apiServer.Port = uint16(addr.Port)
		default:
			return fmt.Errorf("unknown address %v", addr)
		}
	}

	log.Ctx(ctx).Debug().Msgf(
		"API server listening for host %s on %s...", apiServer.Address, listener.Addr().String())

	go func() {
		err := apiServer.httpServer.Serve(listener)
		if err == http.ErrServerClosed {
			log.Ctx(ctx).Debug().Msgf(
				"API server closed for host %s on %s.", apiServer.Address, apiServer.httpServer.Addr)
		} else if err != nil {
			log.Ctx(ctx).Err(err).Msg("Api server can't run. Cannot serve client requests!")
		}
	}()

	return nil
}

// Shutdown shuts down the http server
func (apiServer *Server) Shutdown(ctx context.Context) error {
	return apiServer.httpServer.Shutdown(ctx)
}
