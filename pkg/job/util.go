package job

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	ipfs_spec "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
)

const RegexString = "A-Za-z0-9._~!:@,;+-"

func ParseNodeSelector(nodeSelector string) ([]model.LabelSelectorRequirement, error) {
	selector := strings.TrimSpace(nodeSelector)
	if len(selector) == 0 {
		return []model.LabelSelectorRequirement{}, nil
	}
	requirements, err := labels.ParseToRequirements(selector)
	if err != nil {
		return []model.LabelSelectorRequirement{}, fmt.Errorf("failed to parse node selector: %w", err)
	}
	return model.ToLabelSelectorRequirements(requirements...), nil
}

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

func NewNoopStateLoader() StateLoader {
	stateLoader := func(ctx context.Context, id string) (model.JobState, error) {
		return model.JobState{}, nil
	}
	return stateLoader
}

func BuildJobOutputs(ctx context.Context, outputVolumes []string) ([]spec.Storage, error) {
	outputVolumesMap := make(map[string]spec.Storage)
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
				outputVolumesMap[slices[1]].Mount,
				slices[0],
				slices[1],
			)
		}

		// TODO the intention here is to write the data to IPFS. The IPFSStorageSpec expects a CID
		// but we won't know the CID until the data is hashed. We might want a different type for outputs in these cases.
		// said output ought to describe the intention of writing data to IPFS.
		// previous comment suggests:
		/*
			we have a chance to have a kind of storage multiaddress here
			 e.g. --cid ipfs:abc --cid filecoin:efg
		*/
		ipfsSpec := &ipfs_spec.IPFSStorageSpec{CID: cid.Undef}
		strgspec, err := ipfsSpec.AsSpec(slices[0], slices[2])
		if err != nil {
			return nil, err
		}
		// FIXME: if slices[1] has the same value we will be overriding the value in the map on this line.
		// we must enforce uniqueness for output volume names.
		outputVolumesMap[slices[1]] = strgspec
	}

	var returnOutputVolumes []spec.Storage
	for _, storageSpec := range outputVolumesMap {
		returnOutputVolumes = append(returnOutputVolumes, storageSpec)
	}

	return returnOutputVolumes, nil
}

// ShortID shortens a Job ID e.g. `c42603b4-b418-4827-a9ca-d5a43338f2fe` to `c42603b4`
func ShortID(id string) string {
	if len(id) < model.ShortIDLength {
		return id
	}
	return id[:model.ShortIDLength]
}

func ComputeStateSummary(j model.JobState) string {
	var currentJobState model.ExecutionStateType
	executionStates := FlattenExecutionStates(j)
	for i := range executionStates {
		// If any of the executions are reporting completion, or an active state then
		// we should use that as the summary. Without this we will continue to
		// return BidRejected even when an execution has BidAccepted based on the
		// ordering of the ExecutionStateType enum.
		if executionStates[i].State.IsActive() || executionStates[i].State == model.ExecutionStateCompleted {
			return executionStates[i].State.String()
		}

		if executionStates[i].State > currentJobState {
			currentJobState = executionStates[i].State
		}
	}

	stateSummary := currentJobState.String()
	return stateSummary
}

func ComputeResultsSummary(j *model.JobWithInfo) string {
	var resultSummary string
	completedExecutionStates := GetCompletedExecutionStates(j.State)
	if len(completedExecutionStates) == 0 {
		resultSummary = ""
	} else {
		resultSummary = completedExecutionStates[0].PublishedResult.Name
	}
	return resultSummary
}

func ComputeVerifiedSummary(j *model.JobWithInfo) string {
	var verifiedSummary string
	if j.Job.Spec.Verifier == model.VerifierNoop {
		verifiedSummary = ""
	} else {
		desiredExecutionCount := GetJobConcurrency(j.Job)
		verifiedExecutionCount := CountVerifiedExecutionStates(j.State)
		verifiedSummary = fmt.Sprintf("%d/%d", verifiedExecutionCount, desiredExecutionCount)
	}
	return verifiedSummary
}

func GetIPFSPublishedStorageSpec(executionID string, job model.Job, storageType model.StorageSourceType, cid string) model.StorageSpec {
	return model.StorageSpec{
		Name:          "ipfs://" + cid,
		StorageSource: storageType,
		CID:           cid,
		Metadata:      map[string]string{},
	}
}

func GetJobConcurrency(j model.Job) int {
	concurrency := j.Spec.Deal.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	return concurrency
}
