package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func NewRequesterInfo(requesterNodeID string) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		j.Metadata.Requester = model.JobRequester{
			RequesterNodeID: requesterNodeID,
		}
		return true, nil
	}
}
