package job

import (
	"fmt"
	"regexp"

	"github.com/bacalhau-project/bacalhau/pkg/model"
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

func GetIPFSPublishedStorageSpec(executionID string, job model.Job, storageType model.StorageSourceType, cid string) model.StorageSpec {
	return model.StorageSpec{
		Name:          "ipfs://" + cid,
		StorageSource: storageType,
		CID:           cid,
		Metadata:      map[string]string{},
	}
}
