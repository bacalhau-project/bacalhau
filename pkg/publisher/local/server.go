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
)

func NewLocalPublisherServer(ctx context.Context, directory, address string, port int) *LocalPublisherServer {
	return &LocalPublisherServer{
		rootDirectory: directory,
		address:       address,
		port:          port,
	}
}

func (s *LocalPublisherServer) Run(ctx context.Context) {
	fs := http.FileServer(http.Dir(s.rootDirectory))
	mux := http.NewServeMux()
	mux.Handle("/", fs)

	var server *http.Server

	listenTo := fmt.Sprintf("%s:%d", s.address, s.port)
	go func() {
		server = &http.Server{
			Addr:              listenTo,
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			Handler:           mux,
		}

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	log.Ctx(ctx).Info().Msgf("Running local publishing server on %s", listenTo)

	// Wait for cancellation
	<-ctx.Done()

	log.Ctx(ctx).Info().Msgf("Stopping local publishing server on %s", listenTo)

	if err := server.Shutdown(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error calling shutdown on local publishing server")
	}
}
