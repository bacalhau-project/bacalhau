package publicapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// Server configures a node's public REST API.
type Server struct {
	// Router  *echo.Echo
	protocol string
	address  string
	port     uint16

	tLSCertificateFile string
	tLSKeyFile         string

	httpServer http.Server
}

type ServerParams struct {
	fx.In

	Router *echo.Echo
	Config types.ServerConfig
}

func NewServer(lc fx.Lifecycle, p ServerParams) (*Server, error) {
	var (
		tlsConfig *tls.Config
		protocol  = "http"
	)

	if p.Config.AutoCertDomain != "" {
		log.Info().Msgf("Setting up auto-cert for %s", p.Config.AutoCertDomain)

		autoTLSManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(p.Config.AutoCertCache),
			HostPolicy: autocert.HostWhitelist(p.Config.AutoCertDomain),
		}
		tlsConfig = &tls.Config{
			GetCertificate: autoTLSManager.GetCertificate,
			NextProtos:     []string{acme.ALPNProto},
			MinVersion:     tls.VersionTLS12,
		}

		protocol = "https"
	} else {
		if p.Config.TLSCertificateFile != "" && p.Config.TLSKeyFile != "" {
			protocol = "https"
		}
	}

	server := &Server{
		protocol:           protocol,
		address:            p.Config.Address,
		port:               p.Config.Port,
		tLSCertificateFile: p.Config.TLSCertificateFile,
		tLSKeyFile:         p.Config.TLSKeyFile,
		httpServer: http.Server{
			Handler:           p.Router,
			ReadHeaderTimeout: time.Duration(p.Config.ReadHeaderTimeout),
			WriteTimeout:      time.Duration(p.Config.WriteTimeout),
			TLSConfig:         tlsConfig,
		},
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := server.ListenAndServe(ctx); err != nil {
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Shutdown(ctx)
		},
	})
	return server, nil
}

func (s *Server) Port() uint16 {
	return s.port
}

func (s *Server) Address() string {
	return s.address
}

// GetURI returns the HTTP URI that the server is listening on.
func (s *Server) GetURI() *url.URL {
	interpolated := fmt.Sprintf("%s://%s:%d", s.protocol, s.address, s.port)
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
func (s *Server) ListenAndServe(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.address, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if s.port == 0 {
		switch addr := listener.Addr().(type) {
		case *net.TCPAddr:
			s.port = uint16(addr.Port)
		default:
			return fmt.Errorf("unknown address %v", addr)
		}
	}

	log.Ctx(ctx).Debug().Msgf(
		"API server listening for host %s on %s...", s.address, listener.Addr().String())

	go func() {
		var err error

		if s.protocol == "https" {
			err = s.httpServer.ServeTLS(listener, s.tLSCertificateFile, s.tLSKeyFile)
		} else {
			err = s.httpServer.Serve(listener)
		}

		if err == http.ErrServerClosed {
			log.Ctx(ctx).Debug().Msgf(
				"API server closed for host %s on %s.", s.address, s.httpServer.Addr)
		} else if err != nil {
			log.Ctx(ctx).Err(err).Msg("Api server can't run. Cannot serve client requests!")
		}
	}()

	return nil
}

// Shutdown shuts down the http server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
