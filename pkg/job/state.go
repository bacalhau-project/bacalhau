package job

import (
	"github.com/filecoin-project/bacalhau/pkg/executor"
)

type StateLoader func(id string) (executor.JobState, error)
type StateResolver struct {
	job    executor.Job
	loader StateLoader
}

func NewStateResolver(
	job executor.Job,
	stateLoader StateLoader,
) *StateResolver {
	return &StateResolver{
		job:    job,
		loader: stateLoader,
	}
}

func (resolver *StateResolver) StateSummary() (string, error) {
	_, err := resolver.loader(resolver.job.ID)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (resolver *StateResolver) ResultSummary() (string, error) {
	return "", nil
}

// legacy support for jobs with a single shard
func (resolver *StateResolver) GetResultCIDs() ([]string, error) {
	return []string{}, nil
}

func (resolver *StateResolver) Wait() error {
	return nil
}

// // TODO: #259 We need to rename this - what does it mean to be "furthest along" for a job? Closest to final?
// func GetCurrentJobState(states map[string]executor.JobState) (string, executor.JobState) {
// 	// Returns Node Id, JobState

// 	// Combine the list of jobs down to just those that matter
// 	// Strategy here is assuming the following:
// 	// - All created times are the same (we'll choose the biggest, but it shouldn't matter)
// 	// - All Job IDs are the same (we'll use it as the anchor to combine)
// 	// - If a job has all "bid_rejected", then that's the answer for state
// 	// - If a job has anything BUT bid rejected, then that's the answer for state
// 	// - Everything else SHOULD be equivalent, but doesn't matter (really), so we'll just show the
// 	// 	 one that has the non-bid-rejected result.

// 	finalNodeID := ""
// 	finalJobState := executor.JobState{}

// 	for nodeID, jobState := range states {
// 		if finalNodeID == "" {
// 			finalNodeID = nodeID
// 			finalJobState = jobState
// 		} else if JobStateValue(jobState) > JobStateValue(finalJobState) {
// 			// Overwrite any states that are there with a new state - so we only have one
// 			finalNodeID = nodeID
// 			finalJobState = jobState
// 		}
// 	}
// 	return finalNodeID, finalJobState
// }

// func JobStateValue(jobState executor.JobState) int {
// 	return int(executor.JobStateRunning)
// }

// func getJobResult(job executor.Job, state executor.JobState) string {
// 	if state.ResultsID == "" {
// 		return "-"
// 	}
// 	return "/" + strings.ToLower(job.Spec.Verifier.String()) + "/" + state.ResultsID
// }
