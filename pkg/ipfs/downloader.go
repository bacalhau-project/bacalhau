package ipfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

const (
	DownloadVolumesFolderName = "volumes"
	DownloadShardsFolderName  = "shards"
	DownloadFolderPerm        = 0755
)

type IPFSDownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

const DefaultIPFSTimeout time.Duration = 5 * time.Minute

func NewIPFSDownloadSettings() *IPFSDownloadSettings {
	return &IPFSDownloadSettings{
		TimeoutSecs: int(DefaultIPFSTimeout.Seconds()),
		// we leave this blank so the CLI will auto-create a job folder in pwd
		OutputDir:      "",
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
	// these are the outputs named in the job spec
	// we need them so we know which volumes exists
	outputs []model.StorageSpec,
	// these are the published results we have loaded
	// from the api
	publishedShardResults []model.PublishedResult,
	settings IPFSDownloadSettings,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob")
	defer span.End()

	if len(publishedShardResults) == 0 {
		log.Ctx(ctx).Debug().Msg("No results to download")
		return nil
	}

	switch system.GetEnvironment() {
	case system.EnvironmentProd:
		settings.IPFSSwarmAddrs = strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ",")
	case system.EnvironmentTest:
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			log.Ctx(ctx).Warn().Msg("No action (don't use BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentDev:
		// TODO: add more dev swarm addresses?
		if os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES") != "" {
			settings.IPFSSwarmAddrs = os.Getenv("BACALHAU_IPFS_SWARM_ADDRESSES")
		}
	case system.EnvironmentStaging:
		log.Ctx(ctx).Warn().Msg("Staging environment has no IPFS swarm addresses attached")
	}

	// NOTE: we have to spin up a temporary IPFS node as we don't
	// generally have direct access to a remote node's API server.
	n, err := spinUpIPFSNode(ctx, cm, settings.IPFSSwarmAddrs)
	if err != nil {
		return err
	}

	err = loopOverResults(ctx, n, outputs, publishedShardResults, settings)
	if err != nil {
		return err
	}

	return nil
}

func loopOverResults(
	ctx context.Context,
	n *Node,
	outputs []model.StorageSpec,
	publishedShardResults []model.PublishedResult,
	settings IPFSDownloadSettings,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.loopingOverResults")
	defer span.End()

	log.Ctx(ctx).Debug().Msg("Connecting client to new IPFS node...")
	cl, err := n.Client()
	if err != nil {
		return err
	}

	// this is the full path to the top level folder we are writing our results
	// we have already processed this in the case of a default
	// (i.e. the folder named after the job has been created and assigned)
	resultsOutputDir, err := filepath.Abs(settings.OutputDir)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to get absolute path for output dir: %s", err)
		return err
	}

	// loop over each result directory
	// each result is a storage spec representing a single shards output
	// it's "name" and "path" is named after the shard index
	// so we write the shard output to our scratch folder
	// and then merge each outout volume into the global results
	log.Ctx(ctx).Info().Msgf("Found %d result shards, downloading to temporary folder.", len(publishedShardResults))

	// cleanup the download folders from the results folder
	defer func() {
		for _, result := range publishedShardResults {
			tempShardDownloadDir := filepath.Join(resultsOutputDir, result.Data.Name)
			os.RemoveAll(tempShardDownloadDir)
		}
	}()

	// we move all the contents of the output volume to the global results dir
	// for this output volume
	// find $SOURCE_DIR -name '*' -type f -exec mv -f {} $TARGET_DIR \;
	// append all stdout and stderr to a global concatenated log
	// make a directory for the individual shard logs
	// move the stdout, stderr, and exit code to the shard results dir
	for _, result := range publishedShardResults {
		tempShardDownloadDir := filepath.Join(resultsOutputDir, result.Data.Name)
		err := fetchResult(ctx, result, cl, tempShardDownloadDir, settings.TimeoutSecs)
		if err != nil {
			return err
		}

		err = moveResults(ctx, outputs, result, tempShardDownloadDir, resultsOutputDir)
		if err != nil {
			return err
		}
	}

	return nil
}

func spinUpIPFSNode(
	ctx context.Context,
	cm *system.CleanupManager,
	ipfsSwarmAddrs string,
) (*Node, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.DownloadJob.SpinningUpIPFS")
	defer span.End()

	log.Ctx(ctx).Debug().Msg("Spinning up IPFS node...")
	n, err := NewNode(ctx, cm, strings.Split(ipfsSwarmAddrs, ","))
	if err != nil {
		return nil, err
	}
	return n, nil
}

func fetchResult(
	ctx context.Context,
	result model.PublishedResult,
	cl *Client,
	tempShardDownloadDir string,
	timeoutSecs int,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.fetchingResult")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf("Downloading result CID %s '%s' to '%s'...", result.Data.Name, result.Data.CID, tempShardDownloadDir)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(timeoutSecs)))
		defer cancel()

		return cl.Get(innerCtx, result.Data.CID, tempShardDownloadDir)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

// this moves the results for a single shards PublishedResult, it:
// * merges all output volumes
// * moves stdout, stderr and exitCode to shard folder (and renames files with host id)
// * appends stdout, stderr to global logs
func moveResults(
	ctx context.Context,
	outputVolumes []model.StorageSpec,
	result model.PublishedResult,
	// our temp folder we've downloaded the raw shard results into
	tempShardDownloadDir string,
	// the top level job results folder
	resultsOutputDir string,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.movingResults")
	defer span.End()

	err := mergeOutputVolumes(ctx, outputVolumes, tempShardDownloadDir, resultsOutputDir)
	if err != nil {
		return err
	}

	err = appendGlobalLogs(ctx, tempShardDownloadDir, resultsOutputDir)
	if err != nil {
		return err
	}

	err = moveShardLogs(ctx, result, tempShardDownloadDir, resultsOutputDir)
	if err != nil {
		return err
	}

	return nil
}

// merge the output volumes for each shard into a global volume
func mergeOutputVolumes(
	ctx context.Context,
	outputVolumes []model.StorageSpec,
	// where the raw shard results have been downloaded
	tempShardDownloadDir string,
	// the top level results folder we are writing to
	resultsOutputDir string,
) error {
	// merge each shards volumes into a single volume
	for _, outputVolume := range outputVolumes {
		volumeSourceDir := filepath.Join(tempShardDownloadDir, outputVolume.Name)
		volumeOutputDir := filepath.Join(resultsOutputDir, DownloadVolumesFolderName, outputVolume.Name)
		err := os.MkdirAll(volumeOutputDir, os.ModePerm)
		if err != nil {
			return err
		}
		log.Ctx(ctx).Debug().Msgf("Combining shard from output volume '%s' to final location: '%s'", outputVolume.Name, resultsOutputDir)

		moveFunc := func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// If there is an error reading a path, we should continue with the rest
				log.Ctx(ctx).Error().Err(err).Msgf("Error with path %s", path)
				return nil
			}

			if d.IsDir() {
				return nil
			}

			basePath, err := filepath.Rel(volumeSourceDir, path)
			if err != nil {
				return err
			}

			newPath := filepath.Join(volumeOutputDir, basePath)
			log.Ctx(ctx).Debug().Msgf("Move '%s' to '%s'", path, newPath)
			return os.Rename(path, newPath)
		}

		err = filepath.WalkDir(volumeSourceDir, moveFunc)
		if err != nil {
			return err
		}
	}
	return nil
}

// cat and append stdout & stderr to the global log files for the entire job
func appendGlobalLogs(
	ctx context.Context,
	// where the raw shard results have been downloaded
	tempShardDownloadDir string,
	// the top level results folder we are writing to
	resultsOutputDir string,
) error {
	for _, filename := range []string{
		"stdout",
		"stderr",
	} {
		err := appendFile(
			filepath.Join(tempShardDownloadDir, filename),
			filepath.Join(resultsOutputDir, filename),
		)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			// It's not a problem if one of these files isn't present
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

// move the stdout, stderr, and exit code to the shard results dir
func moveShardLogs(
	ctx context.Context,
	result model.PublishedResult,
	// where the raw shard results have been downloaded
	tempShardDownloadDir string,
	// the top level results folder we are writing to
	resultsOutputDir string,
) error {
	// this is the renamed folder "shards/0" that we write node_XXX_stdout, node_XXX_stderr, and node_XXX_exitCode to
	shardOutputFolder := filepath.Join(resultsOutputDir, DownloadShardsFolderName, fmt.Sprintf("%d", result.ShardIndex))
	err := os.MkdirAll(shardOutputFolder, DownloadFolderPerm)
	if err != nil {
		return err
	}

	for _, filename := range []string{
		"stdout",
		"stderr",
		"exitCode",
	} {
		// we prepend each file with the nodeid so we can have a flat folder of all the logs for this shard
		outputFilename := fmt.Sprintf("node_%s_%s", system.GetShortID(result.NodeID), filename)
		err = os.Rename(
			filepath.Join(tempShardDownloadDir, filename),
			filepath.Join(shardOutputFolder, outputFilename),
		)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			// It's not a problem if one of these files isn't present
			continue
		} else if err != nil {
			return err
		}
	}
	return nil
}

// read data from sourcePath and append it to targetPath
// the same as "cat $sourcePath >> $targetPath"
func appendFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	sink, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer sink.Close()

	_, err = io.Copy(sink, source)
	if err != nil {
		return err
	}

	return nil
}
