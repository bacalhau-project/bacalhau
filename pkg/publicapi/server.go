package publicapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Masterminds/semver"
	"golang.org/x/time/rate"

	"github.com/bacalhau-project/bacalhau/pkg/authz"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"

	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const TimeoutMessage = "Server Timeout!"

type ServerParams struct {
	Router             *echo.Echo
	Address            string
	Port               uint16
	HostID             string
	AutoCertDomain     string
	AutoCertCache      string
	TLSCertificateFile string
	TLSKeyFile         string
	Config             Config
	Authorizer         authz.Authorizer
	Headers            map[string]string
}

// Server configures a node's public REST API.
type Server struct {
	Router  *echo.Echo
	Address string
	Port    uint16

	TLSCertificateFile string
	TLSKeyFile         string

	httpServer http.Server
	config     Config
	useTLS     bool
}

//nolint:funlen
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
		"^/version":                   "/api/v1/version",
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

	// set validator
	server.Router.Validator = NewCustomValidator()

	// enable debug mode to get clearer error messages
	// TODO: disable debug mode after we implement our own error handler
	server.Router.Debug = true

	// set middleware
	logLevel, err := zerolog.ParseLevel(params.Config.LogLevel)
	if err != nil {
		return nil, err
	}

	// base middleware before routing
	server.Router.Pre(
		echomiddelware.Rewrite(migrations),
	)

	serverBuildInfo := version.Get()
	serverVersion, err := semver.NewVersion(serverBuildInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine server agent version %w", err)
	}
	middlewareLogger := log.Ctx(logger.ContextWithNodeIDLogger(context.Background(), params.HostID))
	// base middle after routing
	server.Router.Use(
		echomiddelware.Recover(),
		echomiddelware.RequestID(),
		echomiddelware.BodyLimit(server.config.MaxBytesToReadInBody),
		echomiddelware.RateLimiter(
			echomiddelware.NewRateLimiterMemoryStore(rate.Limit(
				params.Config.ThrottleLimit,
			))),
		echomiddelware.TimeoutWithConfig(
			echomiddelware.TimeoutConfig{
				Timeout:      params.Config.RequestHandlerTimeout,
				ErrorMessage: TimeoutMessage,
				Skipper:      middleware.WebsocketSkipper,
			}),

		middleware.Otel(),
		middleware.Authorize(params.Authorizer),
		// sets headers on the server based on provided config
		middleware.ServerHeader(params.Headers),
		// logs request at appropriate error level based on status code
		middleware.RequestLogger(*middlewareLogger, logLevel),
		// logs requests made by clients with different versions than the server
		middleware.VersionNotifyLogger(middlewareLogger, *serverVersion),
	)

	var tlsConfig *tls.Config
	if params.AutoCertDomain != "" {
		log.Ctx(context.TODO()).Debug().Msgf("Setting up auto-cert for %s", params.AutoCertDomain)

		autoTLSManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(params.AutoCertCache),
			HostPolicy: autocert.HostWhitelist(params.AutoCertDomain),
		}
		tlsConfig = &tls.Config{
			GetCertificate: autoTLSManager.GetCertificate,
			NextProtos:     []string{acme.ALPNProto},
			MinVersion:     tls.VersionTLS12,
		}

		server.useTLS = true
	} else {
		server.useTLS = params.TLSCertificateFile != "" && params.TLSKeyFile != ""
	}

	server.TLSCertificateFile = params.TLSCertificateFile
	server.TLSKeyFile = params.TLSKeyFile

	server.httpServer = http.Server{
		Handler:           server.Router,
		ReadHeaderTimeout: server.config.ReadHeaderTimeout,
		ReadTimeout:       server.config.ReadTimeout,
		WriteTimeout:      server.config.WriteTimeout,
		TLSConfig:         tlsConfig,
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

	// Create an error channel
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		var serveErr error

		if apiServer.useTLS {
			serveErr = apiServer.httpServer.ServeTLS(listener, apiServer.TLSCertificateFile, apiServer.TLSKeyFile)
			if serveErr != nil {
				serveErr = fmt.Errorf(
					"failed to start API server with TLS (cert path: %s) (key path: %s): %w",
					apiServer.TLSCertificateFile,
					apiServer.TLSKeyFile,
					serveErr,
				)
			}
		} else {
			serveErr = apiServer.httpServer.Serve(listener)
			if serveErr != nil {
				serveErr = fmt.Errorf(
					"failed to start API server: %w", err)
			}
		}

		if serveErr == http.ErrServerClosed {
			log.Ctx(ctx).Info().Msgf(
				"API server closed for host %s on %s.", apiServer.Address, apiServer.httpServer.Addr)
		} else if serveErr != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Api server can't run. Cannot serve client requests")
			errChan <- serveErr
		}
	}()

	// NB(forrest): There are few cases when http.Serve* will return an error,
	// one of which is when there are issues reading the TLS configuration.
	// In most cases, if the server starts successfully, http.Serve* will block and not return an error.
	// We use a select statement to wait for either an error from errChan or a timeout of 2 seconds.
	// If an error is received within the timeout, it means the server encountered an issue during startup.
	// If no error is received within the timeout, we assume the server started successfully.
	// TODO: call os.Exit(1) or log.Fatal if the server stops running for a reason other than ErrServerClosed
	select {
	case startErr := <-errChan:
		return startErr
	case <-time.After(time.Second * 2):
		log.Ctx(ctx).Info().Msgf(
			"API server listening for host %s on %s", apiServer.Address, listener.Addr().String())
		return nil
	}
}

// Shutdown shuts down the http server
func (apiServer *Server) Shutdown(ctx context.Context) error {
	return apiServer.httpServer.Shutdown(ctx)
}
