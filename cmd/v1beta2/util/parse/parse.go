package parse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
)

const RegexString = "A-Za-z0-9._~!:@,;+-"

func Labels(ctx context.Context, labels []string) ([]string, error) {
	var jobAnnotations []string
	var unSafeAnnotations []string
	for _, a := range labels {
		if job.IsSafeAnnotation(a) && a != "" {
			jobAnnotations = append(jobAnnotations, a)
		} else {
			unSafeAnnotations = append(unSafeAnnotations, a)
		}
	}

	if len(unSafeAnnotations) > 0 {
		log.Ctx(ctx).Error().Msgf("The following labels are unsafe. Labels must fit the regex '/%s/' (and all emjois): %+v",
			RegexString,
			strings.Join(unSafeAnnotations, ", "))
	}
	return jobAnnotations, nil
}

func NodeSelector(nodeSelector string) ([]v1beta2.LabelSelectorRequirement, error) {
	selector := strings.TrimSpace(nodeSelector)
	if len(selector) == 0 {
		return []v1beta2.LabelSelectorRequirement{}, nil
	}
	requirements, err := labels.ParseToRequirements(selector)
	if err != nil {
		return []v1beta2.LabelSelectorRequirement{}, fmt.Errorf("failed to parse node selector: %w", err)
	}
	return v1beta2.ToLabelSelectorRequirements(requirements...), nil
}

func JobOutputs(ctx context.Context, outputVolumes []string) ([]v1beta2.StorageSpec, error) {
	outputVolumesMap := make(map[string]v1beta2.StorageSpec)
	outputVolumes = append(outputVolumes, "outputs:/outputs")

	for _, outputVolume := range outputVolumes {
		slices := strings.Split(outputVolume, ":")
		if len(slices) != 2 || slices[0] == "" || slices[1] == "" {
			msg := fmt.Sprintf("invalid output volume: %s", outputVolume)
			log.Ctx(ctx).Error().Msg(msg)
			return nil, errors.New(msg)
		}

		if _, containsKey := outputVolumesMap[slices[1]]; containsKey {
			log.Ctx(ctx).Warn().Msgf("Output volumes already contain a mapping to '%s:%s'. Replacing it with '%s:%s'.",
				outputVolumesMap[slices[1]].Name,
				outputVolumesMap[slices[1]].Path,
				slices[0],
				slices[1],
			)
		}

		outputVolumesMap[slices[1]] = v1beta2.StorageSpec{
			// we have a chance to have a kind of storage multiaddress here
			// e.g. --cid ipfs:abc --cid filecoin:efg
			StorageSource: v1beta2.StorageSourceIPFS,
			Name:          slices[0],
			Path:          slices[1],
		}
	}

	var returnOutputVolumes []v1beta2.StorageSpec
	for _, storageSpec := range outputVolumesMap {
		returnOutputVolumes = append(returnOutputVolumes, storageSpec)
	}

	return returnOutputVolumes, nil
}
