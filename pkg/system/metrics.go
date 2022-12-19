package system

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const ServerReadHeaderTimeout = 10 * time.Second

// ListenAndServeMetrics serves prometheus metrics on the specified port.
func ListenAndServeMetrics(ctx context.Context, cm *CleanupManager, port int) error {
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Handler:           sm,
		ReadHeaderTimeout: ServerReadHeaderTimeout,
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	if port == 0 {
		switch addr := listener.Addr().(type) {
		case *net.TCPAddr:
			port = addr.Port
		default:
			return fmt.Errorf("unknown address %v", addr)
		}
	}

	cm.RegisterCallback(func() error {
		// We have to use a separate context, rather than the one passed in, as it may have already been
		// canceled and so would prevent us from performing any cleanup work.
		return srv.Shutdown(context.Background())
	})

	log.Ctx(ctx).Debug().Msgf("Starting metrics server on port %d...", port)
	if err := srv.Serve(listener); err != nil {
		if err == http.ErrServerClosed {
			log.Ctx(ctx).Debug().Msg("Metrics server stopped.")
		} else {
			return errors.Wrap(err, "metrics server failed to Serve")
		}
	}

	return nil
}
