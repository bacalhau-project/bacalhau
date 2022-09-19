package job

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/rs/zerolog/log"
)

const RegexString = "A-Za-z0-9._~!:@,;+-"

func SafeStringStripper(s string) string {
	rChars := SafeAnnotationRegex()
	return rChars.ReplaceAllString(s, "")
}

func IsSafeAnnotation(s string) bool {
	matches := SafeAnnotationRegex().FindString(s)
	return matches == ""
}

func SafeAnnotationRegex() *regexp.Regexp {
	r := regexp.MustCompile(fmt.Sprintf("[^%s|^%s]", returnAllEmojiString(), RegexString))
	return r
}

func NewNoopJobLoader() JobLoader {
	jobLoader := func(ctx context.Context, id string) (model.Job, error) {
		return model.Job{}, nil
	}
	return jobLoader
}

func NewNoopStateLoader() StateLoader {
	stateLoader := func(ctx context.Context, id string) (model.JobState, error) {
		return model.JobState{}, nil
	}
	return stateLoader
}

func buildJobInputs(inputVolumes, inputUrls []string) ([]model.StorageSpec, error) {
	jobInputs := []model.StorageSpec{}

	for _, inputURL := range inputUrls {
		// split using LastIndex to support port numbers in URL
		lastInd := strings.LastIndex(inputURL, ":")
		rawURL := inputURL[:lastInd]
		path := inputURL[lastInd+1:]
		// should loop through all available storage providers?
		_, err := urldownload.IsURLSupported(rawURL)
		if err != nil {
			return []model.StorageSpec{}, err
		}
		jobInputs = append(jobInputs, model.StorageSpec{
			Engine: model.StorageSourceURLDownload,
			URL:    rawURL,
			Path:   path,
		})
	}

	for _, inputVolume := range inputVolumes {
		slices := strings.Split(inputVolume, ":")
		if len(slices) != 2 {
			return []model.StorageSpec{}, fmt.Errorf("invalid input volume: %s", inputVolume)
		}
		jobInputs = append(jobInputs, model.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: model.StorageSourceIPFS,
			CID:    slices[0],
			Path:   slices[1],
		})
	}
	return jobInputs, nil
}

func buildJobOutputs(outputVolumes []string) ([]model.StorageSpec, error) {
	outputVolumesMap := make(map[string]model.StorageSpec)
	outputVolumes = append(outputVolumes, "outputs:/outputs")

	for _, outputVolume := range outputVolumes {
		slices := strings.Split(outputVolume, ":")
		if len(slices) != 2 || slices[0] == "" || slices[1] == "" {
			msg := fmt.Sprintf("invalid output volume: %s", outputVolume)
			log.Error().Msgf(msg)
			return nil, errors.New(msg)
		}

		if _, containsKey := outputVolumesMap[slices[1]]; containsKey {
			log.Warn().Msgf("Output volumes already contain a mapping to '%s:%s'. Replacing it with '%s:%s'.",
				outputVolumesMap[slices[1]].Name,
				outputVolumesMap[slices[1]].Path,
				slices[0],
				slices[1],
			)
		}

		outputVolumesMap[slices[1]] = model.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			Engine: model.StorageSourceIPFS,
			Name:   slices[0],
			Path:   slices[1],
		}
	}

	returnOutputVolumes := []model.StorageSpec{}
	for _, storageSpec := range outputVolumesMap {
		returnOutputVolumes = append(returnOutputVolumes, storageSpec)
	}

	return returnOutputVolumes, nil
}
