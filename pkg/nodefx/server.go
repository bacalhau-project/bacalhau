package nodefx

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
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
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/middleware"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

// Server configures a node's public REST API.
type Server struct {
	Address string
	Port    uint16

	TLSCertificateFile string
	TLSKeyFile         string

	httpServer http.Server
	useTLS     bool
}

func NewAPIServer(lc fx.Lifecycle, cfg *NodeConfig, r *echo.Echo) (*Server, error) {
	server := &Server{
		Address: cfg.ServerConfig.Address,
		Port:    cfg.ServerConfig.Port,
	}

	var tlsConfig *tls.Config
	if cfg.ServerConfig.AutoCertDomain != "" {
		log.Debug().Msgf("Setting up auto-cert for %s", cfg.ServerConfig.AutoCertDomain)

		autoTLSManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(cfg.ServerConfig.AutoCertCache),
			HostPolicy: autocert.HostWhitelist(cfg.ServerConfig.AutoCertDomain),
		}
		tlsConfig = &tls.Config{
			GetCertificate: autoTLSManager.GetCertificate,
			NextProtos:     []string{acme.ALPNProto},
			MinVersion:     tls.VersionTLS12,
		}

		server.useTLS = true
	} else {
		server.useTLS = cfg.ServerConfig.TLSCertificateFile != "" && cfg.ServerConfig.TLSKeyFile != ""
	}

	server.TLSCertificateFile = cfg.ServerConfig.TLSCertificateFile
	server.TLSKeyFile = cfg.ServerConfig.TLSKeyFile

	server.httpServer = http.Server{
		Handler:           r,
		ReadHeaderTimeout: cfg.ServerConfig.ReadHeaderTimeout,
		ReadTimeout:       cfg.ServerConfig.ReadTimeout,
		WriteTimeout:      cfg.ServerConfig.WriteTimeout,
		TLSConfig:         tlsConfig,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%d", cfg.ServerConfig.Address, cfg.ServerConfig.Port)
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			if cfg.ServerConfig.Port == 0 {
				switch addr := listener.Addr().(type) {
				case *net.TCPAddr:
					cfg.ServerConfig.Port = uint16(addr.Port)
				default:
					return fmt.Errorf("unknown address %v", addr)
				}
			}

			log.Ctx(ctx).Debug().Msgf(
				"API server listening for host %s on %s...", cfg.ServerConfig.Address, listener.Addr().String())

			go func() {
				var err error

				if server.useTLS {
					err = server.httpServer.ServeTLS(listener, cfg.ServerConfig.TLSCertificateFile, cfg.ServerConfig.TLSKeyFile)
				} else {
					err = server.httpServer.Serve(listener)
				}

				if err == http.ErrServerClosed {
					log.Ctx(ctx).Debug().Msgf(
						"API server closed for host %s on %s.", cfg.ServerConfig.Address, server.httpServer.Addr)
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

func NewEchoRouter(cfg *NodeConfig, authorizer authz.Authorizer) (*echo.Echo, error) {
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
	mw = append(mw, InitEchoMiddleware(cfg.EchoRouterConfig.EchoMiddlewareConfig)...)
	mw = append(mw, InitTelemetryMiddleware(cfg.EchoRouterConfig.TelemetryMiddlewareConfig)...)
	mw = append(mw, InitServerHeadersMiddleware(cfg.EchoRouterConfig.Headers)...)
	mw = append(mw, InitAuthorizationMiddleware(authorizer)...)

	serverBuildInfo := version.Get()
	serverVersion, err := semver.NewVersion(serverBuildInfo.GitVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to determine server agent version %w", err)
	}
	mw = append(mw, middleware.VersionNotifyLogger(&cfg.EchoRouterConfig.TelemetryMiddlewareConfig.Logger, *serverVersion))
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
