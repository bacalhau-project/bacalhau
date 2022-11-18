package system

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const ServerReadHeaderTimeout = 10 * time.Second

// ListenAndServeMetrics serves prometheus metrics on the specified port.
func ListenAndServeMetrics(ctx context.Context, cm *CleanupManager, port int) error {
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           sm,
		ReadHeaderTimeout: ServerReadHeaderTimeout,
	}

	cm.RegisterCallback(func() error {
		// We have to use a separate context, rather than the one passed in, as it may have already been
		// canceled and so would prevent us from performing any cleanup work.
		return srv.Shutdown(context.Background())
	})

	log.Ctx(ctx).Debug().Msgf("Starting metrics server on port %d...", port)
	if err := srv.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Ctx(ctx).Debug().Msg("Metrics server stopped.")
		} else {
			return fmt.Errorf("metrics server failed to ListenAndServe: %w", err)
		}
	}

	return nil
}
