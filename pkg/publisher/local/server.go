package local

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/network"
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
		address:       resolveAddress(ctx, address),
		port:          port,
	}
}

func resolveAddress(ctx context.Context, address string) string {
	addressType, ok := network.AddressTypeFromString(address)
	if !ok {
		return address
	}

	// If we were provided with an address type and not an address, so we should look up
	// an address from the type.
	addrs, err := network.GetNetworkAddress(addressType, network.AllAddresses)
	if err == nil && len(addrs) > 0 {
		return addrs[0]
	}

	log.Ctx(ctx).Error().Err(err).Stringer("AddressType", addressType).Msgf("unable to find address for type, using 127.0.0.1")
	return "127.0.0.1"
}

func (s *LocalPublisherServer) Start(ctx context.Context) {
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
