package local

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type LocalPublisherServer struct {
	rootDirectory string
	address       string
	port          int
}

const (
	readHeaderTimeout = 3 * time.Second
	readTimeout       = 3 * time.Second
	allAddresses      = "0.0.0.0"
)

func NewLocalPublisherServer(ctx context.Context, directory string, port int) *LocalPublisherServer {
	return &LocalPublisherServer{
		rootDirectory: directory,
		address:       allAddresses, // we listen on all addresses
		port:          port,
	}
}

func (s *LocalPublisherServer) Run(ctx context.Context) {
	fs := http.FileServer(http.Dir(s.rootDirectory))
	mux := http.NewServeMux()
	mux.Handle("/", fs)

	errChan := make(chan error, 1)
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.address, s.port),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		Handler:           mux,
	}

	go func(svr *http.Server, errs chan error) {
		err := svr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}(server, errChan)

	log.Ctx(ctx).Debug().Msgf("Running local publishing server on %s", server.Addr)

	// Wait for cancellation or an error during ListenAndServe
	select {
	case <-ctx.Done(): // context cancelled
		log.Ctx(ctx).Debug().Msg("Shutting down local publishing server")
	case err := <-errChan:
		log.Ctx(ctx).Error().Err(err).Msg("error running local publishing server")
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error calling shutdown on local publishing server")
	}
}
