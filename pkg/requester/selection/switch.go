package selection

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
)

type switcher struct {
	selectors map[bool]requester.NodeSelector
}

// Returns a node selector that switches between node selectors dependent on the
// targeting type defined in the job.
func NewNodeSelectorSwitch(forAny, forAll requester.NodeSelector) requester.NodeSelector {
	return &switcher{
		selectors: map[bool]requester.NodeSelector{
			false: forAny,
			true:  forAll,
		},
	}
}

func (s *switcher) SelectorForJob(job *model.Job) requester.NodeSelector {
	return s.selectors[job.Spec.Deal.TargetAll]
}

// CanCompleteJob implements requester.NodeSelector.
func (s *switcher) CanCompleteJob(ctx context.Context, job *model.Job, jobState *model.JobState) (bool, model.JobStateType) {
	return s.SelectorForJob(job).CanCompleteJob(ctx, job, jobState)
}

// CanVerifyJob implements requester.NodeSelector.
func (s *switcher) CanVerifyJob(ctx context.Context, job *model.Job, jobState *model.JobState) bool {
	return s.SelectorForJob(job).CanVerifyJob(ctx, job, jobState)
}

// SelectBids implements requester.NodeSelector.
func (s *switcher) SelectBids(ctx context.Context, job *model.Job, jobState *model.JobState) (accept, reject []model.ExecutionState) {
	return s.SelectorForJob(job).SelectBids(ctx, job, jobState)
}

// SelectNodes implements requester.NodeSelector.
func (s *switcher) SelectNodes(ctx context.Context, job *model.Job) ([]model.NodeInfo, error) {
	return s.SelectorForJob(job).SelectNodes(ctx, job)
}

// SelectNodesForRetry implements requester.NodeSelector.
func (s *switcher) SelectNodesForRetry(ctx context.Context, job *model.Job, jobState *model.JobState) ([]model.NodeInfo, error) {
	return s.SelectorForJob(job).SelectNodesForRetry(ctx, job, jobState)
}

var _ requester.NodeSelector = (*switcher)(nil)
