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
	cp "github.com/n-marshall/go-cp"
	"github.com/rs/zerolog/log"
)

const (
	DownloadVolumesFolderName = "combined_results"
	DownloadShardsFolderName  = "per_shard"
	DownloadCIDsFolderName    = "raw"
	DownloadFilenameStdout    = "stdout"
	DownloadFilenameStderr    = "stderr"
	DownloadFilenameExitCode  = "exitCode"
	DownloadFolderPerm        = 0755
	DownloadFilePerm          = 0644
)

// SpecialFiles - i.e. aything that is not a volume
// the boolean value is whether we should append to the global log
var SpecialFiles = map[string]bool{
	DownloadFilenameStdout:   true,
	DownloadFilenameStderr:   true,
	DownloadFilenameExitCode: false,
}

type IPFSDownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

type shardCIDContext struct {
	result         model.PublishedResult
	outputVolumes  []model.StorageSpec
	rootDir        string
	cidDownloadDir string
	shardDir       string
	volumeDir      string
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
	outputVolumes []model.StorageSpec,
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

	log.Ctx(ctx).Debug().Msg("Connecting client to new IPFS node...")
	ipfsClient := n.Client()

	// this is the full path to the top level folder we are writing our results
	// we have already processed this in the case of a default
	// (i.e. the folder named after the job has been created and assigned)
	resultsOutputDir, err := filepath.Abs(settings.OutputDir)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to get absolute path for output dir: %s", err)
		return err
	}

	if _, err = os.Stat(resultsOutputDir); os.IsNotExist(err) {
		return fmt.Errorf("output dir does not exist: %s", resultsOutputDir)
	}

	err = os.MkdirAll(filepath.Join(resultsOutputDir, DownloadCIDsFolderName), DownloadFolderPerm)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("Found %d result shards, downloading to: %s.", len(publishedShardResults), resultsOutputDir)

	// each shard context understands the various folder paths
	// and other data it needs to download and resolve itself
	shardContexts := []shardCIDContext{}
	// keep track of which cids we have downloaded to avoid
	// downloading the same cid multiple times
	downloadedCids := map[string]bool{}

	// the base folder for globally merged volumes
	volumeDir := filepath.Join(resultsOutputDir, DownloadVolumesFolderName)
	err = os.Mkdir(volumeDir, DownloadFolderPerm)
	if err != nil {
		return err
	}

	// ensure we have each of the top level merged volumes
	for _, outputVolume := range outputVolumes {
		err = os.MkdirAll(filepath.Join(volumeDir, outputVolume.Name), DownloadFolderPerm)
		if err != nil {
			return err
		}
	}

	// loop over shard results - create their cid and shard folders
	// then add to an array of contexts
	for _, shardResult := range publishedShardResults {
		cidDownloadDir := filepath.Join(resultsOutputDir, DownloadCIDsFolderName, shardResult.Data.CID)
		shardDir := filepath.Join(
			resultsOutputDir,
			DownloadShardsFolderName,
			fmt.Sprintf("%d_node_%s", shardResult.ShardIndex, system.GetShortID(shardResult.NodeID)),
		)
		shardContexts = append(shardContexts, shardCIDContext{
			result:         shardResult,
			outputVolumes:  outputVolumes,
			rootDir:        resultsOutputDir,
			cidDownloadDir: cidDownloadDir,
			shardDir:       shardDir,
			volumeDir:      volumeDir,
		})
	}

	// loop over each result set and download it's CID
	// (if we have not already done so)
	for _, shardContext := range shardContexts {
		_, ok := downloadedCids[shardContext.result.Data.CID]
		if !ok {
			err = fetchResult(ctx, ipfsClient, shardContext, settings.TimeoutSecs)
			if err != nil {
				return err
			}
			downloadedCids[shardContext.result.Data.CID] = true
		}
	}

	// now that we have downloaded the unique CIDs of the results
	// we want to re-construct folders for each shard and volume
	for _, shardContext := range shardContexts {
		err = moveShardData(ctx, shardContext)
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
	cl Client,
	shardContext shardCIDContext,
	timeoutSecs int,
) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/ipfs.fetchingResult")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			shardContext.result.Data.Name,
			shardContext.result.Data.CID, shardContext.cidDownloadDir,
		)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(timeoutSecs)))
		defer cancel()

		return cl.Get(innerCtx, shardContext.result.Data.CID, shardContext.cidDownloadDir)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

func moveShardData(
	ctx context.Context,
	shardContext shardCIDContext,
) error {
	err := os.MkdirAll(shardContext.shardDir, DownloadFolderPerm)
	if err != nil {
		return err
	}

	// the recursive function that will scan our source volume folder
	moveFunc := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// If there is an error reading a path, we should continue with the rest
			log.Ctx(ctx).Error().Err(err).Msgf("Error with path %s", path)
			return nil
		}

		// the relative path of the file/folder
		basePath, err := filepath.Rel(shardContext.cidDownloadDir, path)
		if err != nil {
			return err
		}

		// if we are dealing with the root folder then pass
		// we've already sorted that out above
		if basePath == "." {
			// we don't want to move the root dir
			return nil
		}

		// the path to where we are saving this item in the shard and global folders
		shardTargetPath := filepath.Join(shardContext.shardDir, basePath)
		globalTargetPath := filepath.Join(shardContext.volumeDir, basePath)

		// are we dealing with a special case file?
		shouldAppendLogs, isSpecialFile := SpecialFiles[basePath]

		if d.IsDir() {
			err = os.MkdirAll(shardTargetPath, DownloadFolderPerm)
			if err != nil {
				return nil
			}
			err = os.MkdirAll(globalTargetPath, DownloadFolderPerm)
			if err != nil {
				return nil
			}
		} else {
			// we always copy the file into the shard dir
			err = copyFile(
				path,
				shardTargetPath,
			)
			if err != nil {
				return nil
			}

			// if it's not a special file then we also copy it into the global dir
			if !isSpecialFile {
				err = copyFile(
					path,
					globalTargetPath,
				)
				if err != nil {
					return nil
				}
			}

			// append to the global logs if we should
			if shouldAppendLogs {
				err = appendFile(
					path,
					globalTargetPath,
				)
				if err != nil {
					return nil
				}
			}
		}

		return nil
	}

	err = filepath.WalkDir(shardContext.cidDownloadDir, moveFunc)
	if err != nil {
		return err
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

	sink, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, DownloadFilePerm)
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

func copyFile(sourcePath, targetPath string) error {
	_, err := os.Stat(targetPath)
	if err != nil {
		// we got some other type of error
		if !os.IsNotExist(err) {
			return err
		}
		// file doesn't exist
	} else {
		// this means there was no error and so the file exists
		return nil
	}

	return cp.CopyFile(
		sourcePath,
		targetPath,
	)
}
