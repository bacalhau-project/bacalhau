package jobtransform

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

func NewRequesterInfo(requesterNodeID string, requesterPubKey model.PublicKey) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		j.Metadata.Requester = model.JobRequester{
			RequesterNodeID:    requesterNodeID,
			RequesterPublicKey: requesterPubKey,
		}
		return true, nil
	}
}
