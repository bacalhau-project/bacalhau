package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// specialFiles - i.e. anything that is not a volume
// the boolean value is whether we should append to the global log
var specialFiles = map[string]bool{
	DownloadFilenameStdout:   true,
	DownloadFilenameStderr:   true,
	DownloadFilenameExitCode: true,
}

// DownloadResults downloads published results from a storage source and saves
// them to the specific download path. It supports downloading multiple results
// from different jobs and will append the logs to the global log file. This
// behavior is left from when we supported sharded jobs, and multiple results
// per job. It is not currently being user, and we can evaluate removing it in
// the future if we don't expose merging results from multiple jobs.
//
// * make a temp dir
// * download all results into temp dir
// * ensure top level output dir exists
// * iterate over each published result
// * copy stdout, stderr, exitCode
// * append stdout, stderr to global log
// * iterate over each output volume
// * make new folder for output volume
// * iterate over each result and merge files in output folder to results dir
func DownloadResults( //nolint:funlen
	ctx context.Context,
	publishedResults []*models.SpecConfig,
	downloadProvider DownloaderProvider,
	settings *DownloaderSettings,
) error {
	ctx, cancelFunc := context.WithTimeout(ctx, settings.Timeout)
	defer cancelFunc()

	if len(publishedResults) == 0 {
		log.Ctx(ctx).Debug().Msg("No results to download")
		return nil
	}

	// this is the full path to the top level folder we are writing our results
	// to. We have already processed this in the case of a default
	// (i.e. the folder named after the job has been created and assigned)
	resultsOutputDir, err := filepath.Abs(settings.OutputDir)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to get absolute path for output dir: %s", err)
		return err
	}

	if _, err = os.Stat(resultsOutputDir); os.IsNotExist(err) {
		return fmt.Errorf("output dir does not exist: %s", resultsOutputDir)
	}

	// rawParentDir is the target folder for downloads before they are moved into
	// the end folder at resultsOutputDir. This is typically a directory inside the
	// target directory.
	rawParentDir := filepath.Join(resultsOutputDir, DownloadRawFolderName)
	err = os.MkdirAll(rawParentDir, DownloadFolderPerm)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("Downloading %d results to: %s.", len(publishedResults), resultsOutputDir)
	downloadedResults := make(map[string]struct{})
	for _, publishedResult := range publishedResults {
		downloader, err := downloadProvider.Get(ctx, publishedResult.Type)
		if err != nil {
			return err
		}
		resultPath, err := downloader.FetchResult(ctx, DownloadItem{
			Result:     publishedResult,
			SingleFile: settings.SingleFile,
			ParentPath: rawParentDir,
		})
		if err != nil {
			return err
		}
		downloadedResults[filepath.Clean(resultPath)] = struct{}{}
	}

	if settings.Raw {
		return nil
	}
	for resultPath := range downloadedResults {
		log.Ctx(ctx).Debug().
			Str("Source", resultPath).
			Str("Target", resultsOutputDir).
			Msg("Copying downloaded data to target")

		// if the result is a tar.gz file, we uncompress it first to a folder with the same name (minus the extension)
		// TODO: We could also do this using the content-type for the download (for _some_ downloaders).
		if strings.HasSuffix(resultPath, ".tar.gz") || strings.HasSuffix(resultPath, ".tgz") {
			newResultPath := strings.TrimSuffix(resultPath, ".tar.gz")
			newResultPath = strings.TrimSuffix(newResultPath, ".tgz")

			log.Ctx(ctx).Debug().Msgf("Decompressing %s to %s", resultPath, newResultPath)

			if _, err := os.Stat(newResultPath); os.IsNotExist(err) {
				err = os.MkdirAll(newResultPath, DownloadFolderPerm)
				if err != nil {
					return errors.Wrap(err, "failed to create folder for uncompressed result")
				}
			}

			if err = gzip.Decompress(resultPath, newResultPath); err != nil {
				return err
			}

			resultPath = newResultPath
		}

		err = moveData(ctx, resultPath, resultsOutputDir, len(downloadedResults) > 1)
		if err != nil {
			return err
		}
	}

	return os.RemoveAll(rawParentDir)
}

func moveData(
	ctx context.Context,
	fromFolder string,
	toFolder string,
	appendMode bool,
) error {
	log.Ctx(ctx).Debug().Msgf("Moving data from %s to %s", fromFolder, toFolder)
	// the recursive function that will scan our source volume folder
	moveFunc := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// If there is an error reading a path, we should continue with the rest
			log.Ctx(ctx).Error().Err(err).Msgf("Error with path %s", path)
			return err
		}

		// the relative path of the file/folder
		basePath, err := filepath.Rel(fromFolder, path)
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
		globalTargetPath := filepath.Join(toFolder, basePath)
		log.Ctx(ctx).Debug().
			Str("Source", path).
			Str("BasePath", basePath).
			Str("Target", globalTargetPath).
			Msg("Moving file or directory")

		// are we dealing with a special case file?
		shouldAppendLogs, isSpecialFile := specialFiles[basePath]

		if d.IsDir() {
			err = os.MkdirAll(globalTargetPath, DownloadFolderPerm)
			if err != nil {
				return err
			}
		} else {
			// if it's not a special file then we move it into the global dir
			if !appendMode || !isSpecialFile {
				if err = moveFile(path, globalTargetPath); err != nil {
					return err
				}
			}

			// if this is a special file, and we are in append mode (such as when downloading multiple results), then we
			// append the content instead of overwriting it
			if appendMode && shouldAppendLogs {
				if err = appendFile(path, globalTargetPath); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return filepath.WalkDir(fromFolder, moveFunc)
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

func moveFile(sourcePath, targetPath string) error {
	_, err := os.Stat(targetPath)
	if err != nil {
		// we got some other type of error
		if !os.IsNotExist(err) {
			return err
		}
		// file doesn't exist
	} else {
		return fmt.Errorf(
			"cannot merge results as output already exists: %s. Try --raw to download raw results instead of merging them", targetPath)
	}

	return os.Rename(sourcePath, targetPath)
}
