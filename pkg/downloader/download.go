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

// DownloadResult downloads published results from a storage source and saves
// them to the specific download path. It supports downloading multiple results
// from different jobs and will append the logs to the global log file. This
// behavior is left from when we supported sharded jobs, and multiple results
// per job. It is not currently being user, and we can evaluate removing it in
// the future if we don't expose merging results from multiple jobs.
//
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

	// cidParentDir is the target folder for downloads before they are moved into
	// the end folder at resultsOutputDir. This is typically a directory inside the
	// target directory.
	cidParentDir := filepath.Join(resultsOutputDir, model.DownloadCIDsFolderName)
	err = os.MkdirAll(cidParentDir, model.DownloadFolderPerm)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("Downloading %d results to: %s.", len(publishedResults), resultsOutputDir)

	// keep track of which cids we have downloaded to avoid
	// downloading the same cid multiple times
	downloadedCids := map[string]string{}
	var downloader Downloader

	if settings.SingleFile != "" {
		for _, publishedResult := range publishedResults {
			downloader, err = downloadProvider.Get(ctx, publishedResult.Data.StorageSource) //nolint
			if err != nil {
				return err
			}

			cid, err := findSingleEntry(ctx, publishedResult, downloader, settings.SingleFile)
			if err != nil {
				return err
			}

			// We need to make sure the target folder for this file exists so that we can
			// write to it.
			targetFile := filepath.Join(cidParentDir, settings.SingleFile)
			targetDir := filepath.Dir(targetFile)
			if len(targetDir) > len(cidParentDir) {
				log.Ctx(ctx).Debug().
					Str("Folder", targetDir).
					Msg("creating target folder for single file download")
				err = os.MkdirAll(targetDir, model.DownloadFolderPerm)
				if err != nil {
					log.Ctx(ctx).
						Debug().
						Str("Folder", targetDir).
						Msg("failed to create folder for single file download")
					return err
				}
			}

			// We want to specify the target directory to copy from as the key
			// but the DownloadItem itself specifies the target file to be
			// written to.
			item := model.DownloadItem{
				Name:       settings.SingleFile,
				CID:        cid,
				SourceType: publishedResult.Data.StorageSource,
				Target:     targetFile,
			}

			err = downloader.FetchResult(ctx, item)
			if err != nil {
				return err
			}

			downloadedCids[item.CID] = cidParentDir
		}
	} else {
		for _, publishedResult := range publishedResults {
			downloader, err = downloadProvider.Get(ctx, publishedResult.Data.StorageSource) //nolint
			if err != nil {
				return err
			}

			cidDownloadDir := filepath.Join(cidParentDir, publishedResult.Data.CID)
			_, alreadyExists := downloadedCids[publishedResult.Data.CID]
			if alreadyExists {
				// We don't want to download the same CID twice, so we will just move
				// on to the next item
				log.Ctx(ctx).Debug().
					Str("CID", publishedResult.Data.CID).
					Msg("asked to download a CID a second time")
				continue
			}

			item := model.DownloadItem{
				Name:       publishedResult.Data.Name,
				CID:        publishedResult.Data.CID,
				SourceType: publishedResult.Data.StorageSource,
				Target:     cidDownloadDir,
			}

			err = downloader.FetchResult(ctx, item)
			if err != nil {
				return err
			}

			downloadedCids[item.CID] = cidDownloadDir
		}
	}

	if settings.Raw {
		return nil
	} else {
		// for since file cidDownloadDir is parentid, otherwise it is a cid folder
		for ident, cidDownloadDir := range downloadedCids {
			log.Ctx(ctx).Debug().
				Str("CID", ident).
				Str("Source", cidDownloadDir).
				Str("Target", resultsOutputDir).
				Msg("Copying downloaded data to target")

			err = moveData(ctx, cidDownloadDir, resultsOutputDir, len(downloadedCids) > 1)
			if err != nil {
				return err
			}
		}

		return os.RemoveAll(cidParentDir)
	}
}

func findSingleEntry(ctx context.Context, result model.PublishedResult, downloader Downloader, name string) (string, error) {
	filemap, err := downloader.DescribeResult(ctx, result)
	if err != nil {
		return "", err
	}

	cid, present := filemap[name]
	if !present {
		e := fmt.Errorf("failed to find cid for %s", name)
		log.Ctx(ctx).Error().Err(e).
			Msgf("Finding the CID of %s", name)
		return "", e
	}

	return cid, nil
}

func moveData(
	ctx context.Context,
	fromFolder string,
	toFolder string,
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

		// are we dealing with a special case file?
		shouldAppendLogs, isSpecialFile := specialFiles[basePath]

		if d.IsDir() {
			err = os.MkdirAll(globalTargetPath, model.DownloadFolderPerm)
			if err != nil {
				return err
			}
		} else {
			// if it's not a special file then we move it into the global dir
			if !appendMode || !isSpecialFile {
				err = moveFile(
					path,
					globalTargetPath,
				)
				if err != nil {
					return err
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
		return fmt.Errorf(
			"cannot merge results as output already exists: %s. Try --raw to download raw results instead of merging them", targetPath)
	}

	return os.Rename(sourcePath, targetPath)
}
