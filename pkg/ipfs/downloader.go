package ipfs

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type DownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

func DownloadJob(
	cm *system.CleanupManager,
	job executor.Job,
	results []storage.StorageSpec,
	settings DownloadSettings,
) error {
	if len(results) == 0 {
		log.Debug().Msg("No results to download")
		return nil
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	log.Debug().Msg("Spinning up IPFS node...")
	n, err := NewNode(cm, strings.Split(settings.IPFSSwarmAddrs, ","))
	if err != nil {
		return err
	}

	log.Debug().Msg("Connecting client to new IPFS node...")
	cl, err := n.Client()
	if err != nil {
		return err
	}

	for _, result := range results {
		outputDir := filepath.Join(settings.OutputDir, result.Cid)
		ok, err := system.PathExists(outputDir)
		if err != nil {
			return err
		}
		if ok {
			log.Warn().Msgf("Output directory '%s' already exists, skipping CID '%s'.", outputDir, result.Cid)
			continue
		}

		err = func() error {
			log.Info().Msgf("Downloading result CID '%s' to '%s'...",
				result.Cid, outputDir)

			ctx, cancel := context.WithDeadline(context.Background(),
				time.Now().Add(time.Second*time.Duration(settings.TimeoutSecs)))
			defer cancel()

			return cl.Get(ctx, result.Cid, outputDir)
		}()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Error().Msg("Timed out while downloading result.")
			}

			return err
		}
	}

	return nil
}
