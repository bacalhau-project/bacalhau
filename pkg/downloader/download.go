package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	cp "github.com/n-marshall/go-cp"
	"github.com/rs/zerolog/log"
)

// specialFiles - i.e. anything that is not a volume
// the boolean value is whether we should append to the global log
var specialFiles = map[string]bool{
	model.DownloadFilenameStdout:   true,
	model.DownloadFilenameStderr:   true,
	model.DownloadFilenameExitCode: false,
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
	// these are the outputs named in the job spec
	// we need them so we know which volumes exists
	outputVolumes []model.StorageSpec,
	// these are the published results we have loaded
	// from the api
	publishedShardResults []model.PublishedResult,
	downloadProvider DownloaderProvider,
	settings *model.DownloaderSettings,
) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader.DownloadJob")
	defer span.End()

	if len(publishedShardResults) == 0 {
		log.Ctx(ctx).Debug().Msg("No results to download")
		return nil
	}

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

	err = os.MkdirAll(filepath.Join(resultsOutputDir, model.DownloadCIDsFolderName), model.DownloadFolderPerm)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("Found %d Result shards, downloading to: %s.", len(publishedShardResults), resultsOutputDir)

	// each shard context understands the various folder paths
	// and other data it needs to download and resolve itself
	shardContexts := []shardCIDContext{}
	// keep track of which cids we have downloaded to avoid
	// downloading the same cid multiple times
	downloadedCids := map[string]bool{}

	// the base folder for globally merged volumes
	volumeDir := filepath.Join(resultsOutputDir, model.DownloadVolumesFolderName)
	err = os.Mkdir(volumeDir, model.DownloadFolderPerm)
	if err != nil {
		return err
	}

	// ensure we have each of the top level merged volumes
	for _, outputVolume := range outputVolumes {
		err = os.MkdirAll(filepath.Join(volumeDir, outputVolume.Name), model.DownloadFolderPerm)
		if err != nil {
			return err
		}
	}

	// loop over shard results - create their cid and shard folders
	// then add to an array of contexts
	for _, shardResult := range publishedShardResults {
		cidDownloadDir := filepath.Join(resultsOutputDir, model.DownloadCIDsFolderName, shardResult.Data.CID)
		shardDir := filepath.Join(
			resultsOutputDir,
			model.DownloadShardsFolderName,
			fmt.Sprintf("%d_node_%s", shardResult.ShardIndex, system.GetShortID(shardResult.NodeID)),
		)

		shardContexts = append(shardContexts, shardCIDContext{
			Result:         shardResult,
			OutputVolumes:  outputVolumes,
			RootDir:        resultsOutputDir,
			CIDDownloadDir: cidDownloadDir,
			ShardDir:       shardDir,
			VolumeDir:      volumeDir,
		})

		// get downloader for each shard and download it's CID
		// (if we have not already done so)
		downloader, err := downloadProvider.Get(ctx, shardResult.Data.StorageSource) //nolint
		_, ok := downloadedCids[shardResult.Data.CID]
		if !ok {
			err = downloader.FetchResult(ctx, shardResult, cidDownloadDir)
			if err != nil {
				return err
			}
			downloadedCids[shardResult.Data.CID] = true
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

func moveShardData(
	ctx context.Context,
	shardContext shardCIDContext,
) error {
	err := os.MkdirAll(shardContext.ShardDir, model.DownloadFolderPerm)
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
		basePath, err := filepath.Rel(shardContext.CIDDownloadDir, path)
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
		shardTargetPath := filepath.Join(shardContext.ShardDir, basePath)
		globalTargetPath := filepath.Join(shardContext.VolumeDir, basePath)

		// are we dealing with a special case file?
		shouldAppendLogs, isSpecialFile := specialFiles[basePath]

		if d.IsDir() {
			err = os.MkdirAll(shardTargetPath, model.DownloadFolderPerm)
			if err != nil {
				return nil
			}
			err = os.MkdirAll(globalTargetPath, model.DownloadFolderPerm)
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

	err = filepath.WalkDir(shardContext.CIDDownloadDir, moveFunc)
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

	sink, err := os.OpenFile(targetPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, model.DownloadFilePerm)
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
