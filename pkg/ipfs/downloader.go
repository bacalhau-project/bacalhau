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

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type IPFSDownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

// * make a temp dir
// * download all cids into temp dir
// * ensure top level output dir exists
// * iterate over each shard
// * make new folder for shard logs
// * copy stdout, stderr, exitCode
// * append stdout, stderr to global log
// * iterate over each output volume
// * make new folder for output volume
// * iterate over each shard and merge files in output folder to results dir
func DownloadJob( //nolint:funlen,gocyclo
	cm *system.CleanupManager,
	ctx context.Context,
	job model.Job,
	results []model.StorageSpec,
	settings IPFSDownloadSettings,
) error {
	t := system.GetTracer()

	if len(results) == 0 {
		log.Debug().Msg("No results to download")
		return nil
	}

	finalOutputDirAbs, err := filepath.Abs(settings.OutputDir)
	if err != nil {
		log.Error().Msgf("Failed to get absolute path for output dir: %s", err)
		return err
	}

	spinningUpIPFSCtx, spinningUpIPFSSpan := t.Start(ctx, "spinningupipfs")
	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	log.Debug().Msg("Spinning up IPFS node...")
	n, err := NewNode(cm, spinningUpIPFSCtx, strings.Split(settings.IPFSSwarmAddrs, ","))
	if err != nil {
		return err
	}
	spinningUpIPFSSpan.End()

	log.Debug().Msg("Connecting client to new IPFS node...")
	cl, err := n.Client(ctx)
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
	log.Info().Msgf("Found %d result shards, downloading to temporary folder.", len(results))
	for _, result := range results {
		shardDownloadDir := filepath.Join(scratchFolder, result.Cid)

		err = func() error {
			log.Debug().Msgf("Downloading result CID %s '%s' to '%s'...", result.Name, result.Cid, shardDownloadDir)

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
			volumeOutputDir := filepath.Join(finalOutputDirAbs, "volumes", outputVolume.Name)
			err = os.MkdirAll(volumeOutputDir, os.ModePerm)
			if err != nil {
				return err
			}
			log.Info().Msgf("Combining shard from output volume '%s' to final location: '%s'", outputVolume.Name, finalOutputDirAbs)
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
					filepath.Join(finalOutputDirAbs, filename),
				),
			})
			if err != nil {
				return err
			}
		}

		shardOutputDir := filepath.Join(finalOutputDirAbs, "shards", result.Name)
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
