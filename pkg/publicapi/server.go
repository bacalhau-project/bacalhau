package publicapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Masterminds/semver"
	"golang.org/x/time/rate"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/system"

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

var minClientVersion = semver.MustParse("v1.4.0")

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

	// Register legacy endpoint redirects
	server.registerRedirects(map[string]string{
		"/":        "/api/v1/agent/version",
		"/version": "/api/v1/agent/version",
		"/livez":   "/api/v1/agent/alive",
	})

	// set custom binders and validators
	server.Router.Binder = NewNormalizeBinder()
	server.Router.Validator = NewCustomValidator()

	// enable debug mode to get clearer error messages
	server.Router.Debug = system.IsDebugMode()

	// set middleware
	logLevel, err := zerolog.ParseLevel(params.Config.LogLevel)
	if err != nil {
		return nil, err
	}

	serverBuildInfo := version.Get()
	serverVersion, err := semver.NewVersion(serverBuildInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine server agent version %w", err)
	}
	middlewareLogger := log.Ctx(logger.ContextWithNodeIDLogger(context.Background(), params.HostID))
	// base middle after routing
	server.Router.Use(
		echomiddelware.CORSWithConfig(echomiddelware.CORSConfig{
			AllowOrigins: []string{AllowedCORSOrigin},
		}),
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
		// checks if the client version is supported by the server
		middleware.VersionCheckMiddleware(*serverVersion, *minClientVersion),
		// logs requests made by clients with different versions than the server
		middleware.VersionNotifyLogger(middlewareLogger, *serverVersion),
	)

	// Add custom http error handler. This is a centralized error handler for
	// the server
	server.Router.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

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

// ListenAndServe listens for and serves HTTP requests against the API server.
//

func (apiServer *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", apiServer.Address, apiServer.Port)
	listener, err := net.Listen("tcp", addr) //nolint:noctx // Server lifecycle managed by caller, context used for shutdown
	if err != nil {
		return apiServer.interceptListenError(err)
	}

	if apiServer.Port == 0 {
		switch addr := listener.Addr().(type) {
		case *net.TCPAddr:
			//nolint:gosec // G115: addr.Port should be within limit
			apiServer.Port = uint16(addr.Port)
		default:
			return fmt.Errorf("unknown address %v", addr)
		}
	}

	log.Ctx(ctx).Debug().Msgf(
		"API server listening for host %s on %s...", apiServer.Address, listener.Addr().String())

	go func() {
		var err error

		if apiServer.useTLS {
			err = apiServer.httpServer.ServeTLS(listener, apiServer.TLSCertificateFile, apiServer.TLSKeyFile)
		} else {
			err = apiServer.httpServer.Serve(listener)
		}

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

func (apiServer *Server) interceptListenError(err error) error {
	if strings.Contains(err.Error(), "address already in use") {
		return bacerrors.Newf("address %s is already in use", apiServer.GetURI()).
			WithComponent("APIServer").
			WithCode(bacerrors.ConfigurationError).
			WithHint("To resolve this, either:\n"+
				"1. Check if you are already running bacalhau\n"+
				"2. Stop the other process using the port\n"+
				"3. Configure a different port using one of these methods:\n"+
				"   a. Use the `-c %s=<new_port>` flag with your serve command\n"+
				"   b. Set the port in a configuration file with `%s config set %s=<new_port>`",
				types.APIPortKey, os.Args[0], types.APIPortKey)
	}
	return err
}

// registerRedirects registers GET handlers for each path that redirect to their corresponding destination
func (apiServer *Server) registerRedirects(redirects map[string]string) {
	for path, dest := range redirects {
		apiServer.Router.GET(path, func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, dest)
		})
	}
}
