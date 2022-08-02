package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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

// * make a temp dir
// * download all cids into temp dir
// * ensure top level output dir exists
// * iterate over each shard
//   * make new folder for shard logs
//   * copy stdout, stderr, exitCode
//   * append stdout, stderr to global log
// * iterate over each output volume
//   * make new folder for output volume
//   * iterate over each shard and merge files in output folder to results dir
func DownloadJob( //nolint:funlen,gocyclo
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

	scratchFolder, err := ioutil.TempDir("", "bacalhau-ipfs-job-downloader")
	if err != nil {
		return err
	}
	log.Debug().Msgf("Created download scratch folder: %s", scratchFolder)

	// loop over each result directory
	// each result is a storage spec representing a single shards output
	// it's "name" and "path" is named after the shard index
	// so we write the shard output to our scratch folder
	// and then merge each outout volume into the global results
	for _, result := range results {
		shardDownloadDir := filepath.Join(scratchFolder, result.Cid)

		err = func() error {
			log.Info().Msgf("Downloading result CID %s '%s' to '%s'...", result.Name, result.Cid, shardDownloadDir)

			ctx, cancel := context.WithDeadline(context.Background(),
				time.Now().Add(time.Second*time.Duration(settings.TimeoutSecs)))
			defer cancel()

			return cl.Get(ctx, result.Cid, shardDownloadDir)
		}()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Error().Msg("Timed out while downloading result.")
			}

			return err
		}

		// we move all the contents of the output volume to the global results dir
		// for this output volume
		for _, outputVolume := range job.Spec.Outputs {
			volumeSourceDir := filepath.Join(shardDownloadDir, outputVolume.Name)
			volumeOutputDir := filepath.Join(settings.OutputDir, "volumes", outputVolume.Name)
			err = os.MkdirAll(volumeOutputDir, os.ModePerm)
			if err != nil {
				return err
			}
			log.Info().Msgf("Copying output volume %s", outputVolume.Name)
			// find $SOURCE_DIR -name '*' -type f -exec mv -f {} $TARGET_DIR \;
			err = system.RunCommand("bash", []string{
				"-c",
				fmt.Sprintf("find %s -name '*' -type f -exec mv -f {} %s \\;", volumeSourceDir, volumeOutputDir),
			})
			if err != nil {
				return err
			}
		}

		// append all stdout and stderr to a global concatenated log
		for _, filename := range []string{
			"stdout",
			"stderr",
		} {
			err = system.RunCommand("bash", []string{
				"-c",
				fmt.Sprintf(
					"cat %s >> %s",
					filepath.Join(shardDownloadDir, filename),
					filepath.Join(settings.OutputDir, filename),
				),
			})
			if err != nil {
				return err
			}
		}

		shardOutputDir := filepath.Join(settings.OutputDir, "shards", result.Name)
		// make a directory for the individual shard logs
		err = os.MkdirAll(shardOutputDir, os.ModePerm)
		if err != nil {
			return err
		}

		// move the stdout, stderr, and exit code to the shard results dir
		for _, filename := range []string{
			"stdout",
			"stderr",
			"exitCode",
		} {
			err = system.RunCommand("mv", []string{
				filepath.Join(shardDownloadDir, filename),
				shardOutputDir,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
