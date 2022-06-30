package system

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// ListenAndServeMetrics serves prometheus metrics on the specified port.
func ListenAndServeMetrics(cm *CleanupManager, port int) error {
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())

	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: sm,
	}

	cm.RegisterCallback(func() error {
		return srv.Shutdown(context.Background())
	})

	log.Debug().Msgf("Starting metrics server on port %d...", port)
	if err := srv.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Debug().Msg("Metrics server stopped.")
		} else {
			return fmt.Errorf("metrics server failed to ListenAndServe: %w", err)
		}
	}

	return nil
}
