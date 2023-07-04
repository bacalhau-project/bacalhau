package publicapi

import (
	"context"

	"github.com/gorilla/websocket"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type RequesterAPIClientWrapper struct {
	Client *RequesterAPIClient
}

func (w *RequesterAPIClientWrapper) List(
	ctx context.Context,
	idFilter string,
	includeTags []model.IncludedTag,
	excludeTags []model.ExcludedTag,
	maxJobs int,
	returnAll bool,
	sortBy string,
	sortReverse bool,
) ([]*model.JobWithInfo, error) {
	res, err := w.Client.List(
		ctx,
		idFilter,
		model.ConvertIncludeTagListToV1beta2(includeTags...),
		model.ConvertExcludeTagListToV1beta2(excludeTags...),
		maxJobs,
		returnAll,
		sortBy,
		sortReverse,
	)
	if err != nil {
		return nil, err
	}
	return model.ConvertV1beta2JobWithInfoList(res...), nil
}

func (w *RequesterAPIClientWrapper) Nodes(ctx context.Context) ([]model.NodeInfo, error) {
	res, err := w.Client.Nodes(ctx)
	if err != nil {
		return nil, err
	}
	return model.ConvertV1beta2NodeInfoList(res...), nil
}

func (w *RequesterAPIClientWrapper) Cancel(ctx context.Context, jobID string, reason string) (*model.JobState, error) {
	res, err := w.Client.Cancel(ctx, jobID, reason)
	if err != nil {
		return nil, err
	}
	out := model.ConvertV1beta2JobState(*res)
	return &out, nil
}

func (w *RequesterAPIClientWrapper) Get(ctx context.Context, jobID string) (*model.JobWithInfo, bool, error) {
	res, found, err := w.Client.Get(ctx, jobID)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, found, nil
	}
	return model.ConvertV1beta2JobWithInfo(res), found, nil
}

func (w *RequesterAPIClientWrapper) GetJobState(ctx context.Context, jobID string) (model.JobState, error) {
	res, err := w.Client.GetJobState(ctx, jobID)
	if err != nil {
		return model.JobState{}, err
	}
	return model.ConvertV1beta2JobState(res), nil
}

func (w *RequesterAPIClientWrapper) GetJobStateResolver() *job.StateResolver {
	return w.Client.GetJobStateResolver()
}

func (w *RequesterAPIClientWrapper) GetEvents(
	ctx context.Context,
	jobID string,
	options EventFilterOptions,
) ([]model.JobHistory, error) {
	res, err := w.Client.GetEvents(ctx, jobID, options)
	if err != nil {
		return nil, err
	}
	return model.ConvertV1beta2JobHistoryListToJobHistoryList(res...), nil
}

func (w *RequesterAPIClientWrapper) GetResults(ctx context.Context, jobID string) ([]model.PublishedResult, error) {
	res, err := w.Client.GetResults(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return model.ConvertV1beta2PublishedResultList(res...), nil
}

func (w *RequesterAPIClientWrapper) Submit(ctx context.Context, j *model.Job) (*model.Job, error) {
	in := model.ConvertJobToV1beta2(*j)
	res, err := w.Client.Submit(ctx, &in)
	if err != nil {
		// TODO(forrest): [fixme] we return the passed job because callers of this method want to look at it's ID
		// Submit could just return the ID of a job rather than a mutated version of the submitted job.
		return j, err
	}

	out := model.ConvertV1beta2Job(*res)
	return &out, nil
}

func (w *RequesterAPIClientWrapper) Approve(
	ctx context.Context,
	jobID string,
	response bidstrategy.BidStrategyResponse,
) error {
	return w.Client.Approve(ctx, jobID, response)
}

func (w *RequesterAPIClientWrapper) Logs(
	ctx context.Context,
	jobID string,
	executionID string,
	withHistory bool,
	follow bool,
) (*websocket.Conn, error) {
	return w.Client.Logs(ctx, jobID, executionID, withHistory, follow)
}

func (w *RequesterAPIClientWrapper) Debug(ctx context.Context) (map[string]model.DebugInfo, error) {
	res, err := w.Client.Debug(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]model.DebugInfo)
	for k, v := range res {
		out[k] = model.DebugInfo{
			Component: v.Component,
			Info:      v.Info,
		}
	}
	return out, nil
}
func (w *RequesterAPIClientWrapper) Version(ctx context.Context) (*model.BuildVersionInfo, error) {
	return w.Client.Version(ctx)
}

func (w *RequesterAPIClientWrapper) Alive(ctx context.Context) (bool, error) {
	return w.Client.Alive(ctx)
}
