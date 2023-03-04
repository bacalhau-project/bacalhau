package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// specialFiles - i.e. anything that is not a volume
// the boolean value is whether we should append to the global log
var specialFiles = map[string]bool{
	model.DownloadFilenameStdout:   true,
	model.DownloadFilenameStderr:   true,
	model.DownloadFilenameExitCode: true,
}

// DownloadResult downloads published results from a storage source and saves them to the specific download path.
// It supports downloading multiple results from different jobs and will append the logs to the global log file. This behavior is left
// from when we supported sharded jobs, and multiple results per job. It is not currently being user, and we can evaluate removing it in
// the future if we don't expose merging results from multiple jobs.
// * make a temp dir
// * download all cids into temp dir
// * ensure top level output dir exists
// * iterate over each published result
// * copy stdout, stderr, exitCode
// * append stdout, stderr to global log
// * iterate over each output volume
// * make new folder for output volume
// * iterate over each result and merge files in output folder to results dir
func DownloadResults( //nolint:funlen,gocyclo
	ctx context.Context,
	publishedResults []model.PublishedResult,
	downloadProvider DownloaderProvider,
	settings *model.DownloaderSettings,
) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader.DownloadResults")
	defer span.End()

	if len(publishedResults) == 0 {
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

	cidParentDir := filepath.Join(resultsOutputDir, model.DownloadCIDsFolderName)
	err = os.MkdirAll(cidParentDir, model.DownloadFolderPerm)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("Downloading %d results to: %s.", len(publishedResults), resultsOutputDir)

	// keep track of which cids we have downloaded to avoid
	// downloading the same cid multiple times
	downloadedCids := map[string]string{}

	for _, publishedResult := range publishedResults {
		cidDownloadDir := filepath.Join(cidParentDir, publishedResult.Data.CID)
		_, ok := downloadedCids[publishedResult.Data.CID]
		if !ok {
			downloader, err := downloadProvider.Get(ctx, publishedResult.Data.StorageSource) //nolint
			err = downloader.FetchResult(ctx, publishedResult, cidDownloadDir)
			if err != nil {
				return err
			}
			downloadedCids[publishedResult.Data.CID] = cidDownloadDir
		}
	}

	for _, cidDownloadDir := range downloadedCids {
		err = moveData(ctx, resultsOutputDir, cidDownloadDir, len(downloadedCids) > 1)
		if err != nil {
			return err
		}
	}
	return os.RemoveAll(cidParentDir)
}

func moveData(
	ctx context.Context,
	volumeDir string,
	cidDownloadDir string,
	appendMode bool,
) error {
	// the recursive function that will scan our source volume folder
	moveFunc := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// If there is an error reading a path, we should continue with the rest
			log.Ctx(ctx).Error().Err(err).Msgf("Error with path %s", path)
			return err
		}

		// the relative path of the file/folder
		basePath, err := filepath.Rel(cidDownloadDir, path)
		if err != nil {
			return err
		}

		// if we are dealing with the root folder then pass
		// we've already sorted that out above
		if basePath == "." {
			// we don't want to move the root dir
			return nil
		}

		// the path to where we are saving this item in the global folders
		globalTargetPath := filepath.Join(volumeDir, basePath)

		// are we dealing with a special case file?
		shouldAppendLogs, isSpecialFile := specialFiles[basePath]

		if d.IsDir() {
			err = os.MkdirAll(globalTargetPath, model.DownloadFolderPerm)
			if err != nil {
				return nil
			}
		} else {
			// if it's not a special file then we move it into the global dir
			if !appendMode || !isSpecialFile {
				err = moveFile(
					path,
					globalTargetPath,
				)
				if err != nil {
					return nil
				}
			}

			// if this is a special file, and we are in append mode (such as when downloading multiple results), then we
			// append the content instead of overwriting it
			if appendMode && shouldAppendLogs {
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

	return filepath.WalkDir(cidDownloadDir, moveFunc)
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

func moveFile(sourcePath, targetPath string) error {
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

	return os.Rename(sourcePath, targetPath)
}
