package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Masterminds/semver"
	"github.com/labstack/echo/v4"
	echomiddelware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/time/rate"

	"github.com/bacalhau-project/bacalhau/pkg/authz"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/lib/policy"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

var Module = fx.Module("server",
	fx.Provide(NewAPIServer),
	fx.Provide(NewEchoRouter),
)

// Server configures a node's public REST API.
type Server struct {
	Address  string
	Port     uint16
	Protocol string

	TLSCertificateFile string
	TLSKeyFile         string

	httpServer http.Server
	useTLS     bool
}

func (s *Server) GetURI() *url.URL {
	interpolated := fmt.Sprintf("%s://%s:%d", s.Protocol, s.Address, s.Port)
	url, err := url.Parse(interpolated)
	if err != nil {
		panic(fmt.Errorf("callback url must parse: %s", interpolated))
	}
	return url
}

func NewAPIServer(lc fx.Lifecycle, cfg node.NodeConfig, r *echo.Echo) (*Server, error) {
	authzPolicy, err := policy.FromPathOrDefault(cfg.AuthConfig.AccessPolicyPath, authz.AlwaysAllowPolicy)
	if err != nil {
		return nil, err
	}

	signingKey, err := pkgconfig.GetClientPublicKey()
	if err != nil {
		return nil, err
	}

	serverVersion := version.Get()
	// public http api server
	serverParams := publicapi.ServerParams{
		Address:    cfg.HostAddress,
		Port:       cfg.APIPort,
		HostID:     cfg.NodeID,
		Config:     cfg.APIServerConfig,
		Authorizer: authz.NewPolicyAuthorizer(authzPolicy, signingKey, cfg.NodeID),
		Headers: map[string]string{
			apimodels.HTTPHeaderBacalhauGitVersion: serverVersion.GitVersion,
			apimodels.HTTPHeaderBacalhauGitCommit:  serverVersion.GitCommit,
			apimodels.HTTPHeaderBacalhauBuildDate:  serverVersion.BuildDate.UTC().String(),
			apimodels.HTTPHeaderBacalhauBuildOS:    serverVersion.GOOS,
			apimodels.HTTPHeaderBacalhauArch:       serverVersion.GOARCH,
		},
	}

	// Only allow autocert for requester nodes
	if cfg.IsRequesterNode {
		serverParams.AutoCertDomain = cfg.RequesterAutoCert
		serverParams.AutoCertCache = cfg.RequesterAutoCertCache
		serverParams.TLSCertificateFile = cfg.RequesterTLSCertificateFile
		serverParams.TLSKeyFile = cfg.RequesterTLSKeyFile
	}
	server := &Server{
		Address: cfg.HostAddress,
		Port:    cfg.APIPort,
		// TODO this is mostly unused
		Protocol: "http",
	}

	var tlsConfig *tls.Config
	if cfg.RequesterAutoCert != "" {
		log.Debug().Msgf("Setting up auto-cert for %s", cfg.RequesterAutoCert)

		autoTLSManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(cfg.RequesterAutoCertCache),
			HostPolicy: autocert.HostWhitelist(cfg.RequesterAutoCert),
		}
		tlsConfig = &tls.Config{
			GetCertificate: autoTLSManager.GetCertificate,
			NextProtos:     []string{acme.ALPNProto},
			MinVersion:     tls.VersionTLS12,
		}

		server.useTLS = true
	} else {
		server.useTLS = cfg.RequesterTLSCertificateFile != "" && cfg.RequesterTLSKeyFile != ""
	}

	server.TLSCertificateFile = cfg.RequesterTLSCertificateFile
	server.TLSKeyFile = cfg.RequesterTLSKeyFile

	server.httpServer = http.Server{
		Handler:           r,
		ReadHeaderTimeout: cfg.APIServerConfig.ReadHeaderTimeout,
		ReadTimeout:       cfg.APIServerConfig.ReadTimeout,
		WriteTimeout:      cfg.APIServerConfig.WriteTimeout,
		TLSConfig:         tlsConfig,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%d", cfg.HostAddress, cfg.APIPort)
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			if cfg.APIPort == 0 {
				switch addr := listener.Addr().(type) {
				case *net.TCPAddr:
					cfg.APIPort = uint16(addr.Port)
				default:
					return fmt.Errorf("unknown address %v", addr)
				}
			}

			log.Ctx(ctx).Debug().Msgf(
				"API server listening for host %s on %s...", cfg.HostAddress, listener.Addr().String())

			go func() {
				var err error

				if server.useTLS {
					err = server.httpServer.ServeTLS(listener, cfg.RequesterTLSCertificateFile, cfg.RequesterTLSKeyFile)
				} else {
					err = server.httpServer.Serve(listener)
				}

				if err == http.ErrServerClosed {
					log.Ctx(ctx).Debug().Msgf(
						"API server closed for host %s on %s.", cfg.HostAddress, server.httpServer.Addr)
				} else if err != nil {
					log.Ctx(ctx).Err(err).Msg("Api server can't run. Cannot serve client requests!")
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {
			return server.httpServer.Shutdown(ctx)
		},
	})

	return server, nil
}

func NewEchoRouter(cfg node.NodeConfig, authorizer authz.Authorizer) (*echo.Echo, error) {
	instance := echo.New()
	instance.Validator = publicapi.NewCustomValidator()
	// TODO: disable debug mode after we implement our own error handler
	instance.Debug = true
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
	// base middleware before routing
	instance.Pre(
		echomiddelware.Rewrite(migrations),
	)
	var mw []echo.MiddlewareFunc
	mw = append(mw, InitEchoMiddleware(EchoMiddlewareConfig{
		MaxBytesToReadInBody:  cfg.APIServerConfig.MaxBytesToReadInBody,
		ThrottleLimit:         cfg.APIServerConfig.ThrottleLimit,
		RequestHandlerTimeout: cfg.APIServerConfig.RequestHandlerTimeout,
	})...)
	level, err := zerolog.ParseLevel(cfg.APIServerConfig.LogLevel)
	logger := log.Logger
	if err != nil {
		return nil, err
	}
	mw = append(mw, InitTelemetryMiddleware(TelemetryMiddlewareConfig{
		Logger:   logger,
		LogLevel: level,
	})...)

	serverVersion := version.Get()
	mw = append(mw, InitServerHeadersMiddleware(map[string]string{
		apimodels.HTTPHeaderBacalhauGitVersion: serverVersion.GitVersion,
		apimodels.HTTPHeaderBacalhauGitCommit:  serverVersion.GitCommit,
		apimodels.HTTPHeaderBacalhauBuildDate:  serverVersion.BuildDate.UTC().String(),
		apimodels.HTTPHeaderBacalhauBuildOS:    serverVersion.GOOS,
		apimodels.HTTPHeaderBacalhauArch:       serverVersion.GOARCH,
	})...)
	mw = append(mw, InitAuthorizationMiddleware(authorizer)...)

	serverBuildInfo := version.Get()
	serverSemVersion, err := semver.NewVersion(serverBuildInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine server agent version %w", err)
	}
	mw = append(mw, middleware.VersionNotifyLogger(&logger, *serverSemVersion))
	instance.Use(mw...)

	return instance, nil
}

type EchoMiddlewareConfig struct {
	// MaxBytesToReadInBody is used by safeHandlerFuncWrapper as the max size of body
	MaxBytesToReadInBody string

	// ThrottleLimit is the maximum number of requests per second
	ThrottleLimit int

	// This represents maximum duration for handlers to complete, or else fail the request with 503 error code.
	RequestHandlerTimeout time.Duration
}

func InitEchoMiddleware(cfg EchoMiddlewareConfig) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		echomiddelware.CORS(),
		echomiddelware.Recover(),
		echomiddelware.RequestID(),
		echomiddelware.BodyLimit(cfg.MaxBytesToReadInBody),
		echomiddelware.RateLimiter(
			echomiddelware.NewRateLimiterMemoryStore(rate.Limit(
				cfg.ThrottleLimit,
			))),
		echomiddelware.TimeoutWithConfig(
			echomiddelware.TimeoutConfig{
				Timeout:      cfg.RequestHandlerTimeout,
				ErrorMessage: "Server Timeout",
				Skipper:      middleware.WebsocketSkipper,
			}),
	}
}

type TelemetryMiddlewareConfig struct {
	Logger   zerolog.Logger
	LogLevel zerolog.Level
}

func InitTelemetryMiddleware(cfg TelemetryMiddlewareConfig) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		middleware.Otel(),
		middleware.RequestLogger(cfg.Logger, cfg.LogLevel),
	}
}

func InitAuthorizationMiddleware(authorizer authz.Authorizer) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		middleware.Authorize(authorizer),
	}
}

func InitServerHeadersMiddleware(headers map[string]string) []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		middleware.ServerHeader(headers),
	}
}
