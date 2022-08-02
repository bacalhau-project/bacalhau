package ipfs

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type DownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

func DownloadCIDs(
	cm *system.CleanupManager,
	cids []string,
	settings DownloadSettings,
) error {
	if len(cids) == 0 {
		log.Debug().Msg("No cids to download")
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

	// NOTE: this will run in non-deterministic order
	for _, cid := range cids {
		outputDir := filepath.Join(settings.OutputDir, cid)
		ok, err := system.PathExists(outputDir)
		if err != nil {
			return err
		}
		if ok {
			log.Warn().Msgf("Output directory '%s' already exists, skipping CID '%s'.", outputDir, cid)
			continue
		}

		err = func() error {
			log.Info().Msgf("Downloading result CID '%s' to '%s'...",
				cid, outputDir)

			ctx, cancel := context.WithDeadline(context.Background(),
				time.Now().Add(time.Second*time.Duration(settings.TimeoutSecs)))
			defer cancel()

			return cl.Get(ctx, cid, outputDir)
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
