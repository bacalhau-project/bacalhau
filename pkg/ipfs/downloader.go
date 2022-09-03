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

func NewIPFSDownloadSettings() *IPFSDownloadSettings {
	return &IPFSDownloadSettings{
		TimeoutSecs:    10,
		OutputDir:      ".",
		IPFSSwarmAddrs: "",
	}
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
	ctx context.Context,
	cm *system.CleanupManager,
	job model.Job,
	results []model.StorageSpec,
	settings IPFSDownloadSettings,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob")
	defer span.End()

	if len(results) == 0 {
		log.Debug().Msg("No results to download")
		return nil
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	n, err := spinUpIPFSNode(ctx, cm, settings.IPFSSwarmAddrs)
	if err != nil {
		return err
	}

	err = loopOverResults(ctx, n, results, settings, job)
	if err != nil {
		return err
	}

	return nil
}

func loopOverResults(ctx context.Context,
	n *Node,
	results []model.StorageSpec,
	settings IPFSDownloadSettings,
	job model.Job) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.loopingOverResults")
	defer span.End()

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

	finalOutputDirAbs, err := filepath.Abs(settings.OutputDir)
	if err != nil {
		log.Error().Msgf("Failed to get absolute path for output dir: %s", err)
		return err
	}

	// loop over each result directory
	// each result is a storage spec representing a single shards output
	// it's "name" and "path" is named after the shard index
	// so we write the shard output to our scratch folder
	// and then merge each outout volume into the global results
	log.Info().Msgf("Found %d result shards, downloading to temporary folder.", len(results))

	// we move all the contents of the output volume to the global results dir
	// for this output volume
	// find $SOURCE_DIR -name '*' -type f -exec mv -f {} $TARGET_DIR \;
	// append all stdout and stderr to a global concatenated log
	// make a directory for the individual shard logs
	// move the stdout, stderr, and exit code to the shard results dir
	for _, result := range results {
		shardDownloadDir := filepath.Join(scratchFolder, result.Name)
		err := fetchResult(ctx, result, cl, shardDownloadDir, settings.TimeoutSecs)
		if err != nil {
			return err
		}

		err = moveResults(ctx, job, shardDownloadDir, finalOutputDirAbs, result)
		if err != nil {
			return err
		}
	}
	return nil
}

func spinUpIPFSNode(ctx context.Context,
	cm *system.CleanupManager,
	ipfsSwarmAddrs string) (*Node, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob.SpinningUpIPFS")
	defer span.End()

	log.Debug().Msg("Spinning up IPFS node...")
	n, err := NewNode(ctx, cm, strings.Split(ipfsSwarmAddrs, ","))
	if err != nil {
		return nil, err
	}
	return n, nil
}

func fetchResult(ctx context.Context,
	result model.StorageSpec,
	cl *Client,
	shardDownloadDir string,
	timeoutSecs int) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.fetchingResult")
	defer span.End()

	err := func() error {
		log.Debug().Msgf("Downloading result CID %s '%s' to '%s'...", result.Name, result.Cid, shardDownloadDir)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(timeoutSecs)))
		defer cancel()

		return cl.Get(innerCtx, result.Cid, shardDownloadDir)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

func moveResults(ctx context.Context,
	job model.Job,
	shardDownloadDir string,
	finalOutputDirAbs string,
	result model.StorageSpec) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.movingResults")
	defer span.End()

	for _, outputVolume := range job.Spec.Outputs {
		volumeSourceDir := filepath.Join(shardDownloadDir, outputVolume.Name)
		volumeOutputDir := filepath.Join(finalOutputDirAbs, "volumes", outputVolume.Name)
		err := os.MkdirAll(volumeOutputDir, os.ModePerm)
		if err != nil {
			return err
		}
		log.Info().Msgf("Combining shard from output volume '%s' to final location: '%s'", outputVolume.Name, finalOutputDirAbs)

		err = system.RunCommand("bash", []string{
			"-c",
			fmt.Sprintf("find %s -name '*' -type f -exec mv -f {} %s \\;", volumeSourceDir, volumeOutputDir),
		})
		if err != nil {
			return err
		}
	}

	err := catStdFiles(ctx, shardDownloadDir, finalOutputDirAbs)
	if err != nil {
		return err
	}

	shardOutputDir := filepath.Join(finalOutputDirAbs, "shards", result.Name)

	err = moveStdFiles(ctx, shardDownloadDir, shardOutputDir)
	if err != nil {
		return err
	}

	return nil
}

func catStdFiles(ctx context.Context,
	shardDownloadDir, finalOutputDirAbs string) error {
	for _, filename := range []string{
		"stdout",
		"stderr",
	} {
		err := system.RunCommand("bash", []string{
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
	return nil
}

func moveStdFiles(ctx context.Context,
	shardDownloadDir, shardOutputDir string) error {
	err := os.MkdirAll(shardOutputDir, os.ModePerm)
	if err != nil {
		return err
	}

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
	return nil
}
