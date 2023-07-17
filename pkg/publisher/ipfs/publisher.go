package ipfs

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type IPFSPublisher struct {
	IPFSClient ipfs.Client
}

func NewIPFSPublisher(
	ctx context.Context,
	_ *system.CleanupManager,
	cl ipfs.Client,
) (*IPFSPublisher, error) {
	log.Ctx(ctx).Debug().Msgf("IPFS publisher initialized for node: %s", cl.APIAddress())
	return &IPFSPublisher{
		IPFSClient: cl,
	}, nil
}

func (publisher *IPFSPublisher) IsInstalled(ctx context.Context) (bool, error) {
	_, err := publisher.IPFSClient.ID(ctx)
	return err == nil, err
}

func (publisher *IPFSPublisher) ValidateJob(ctx context.Context, j model.Job) error {
	switch j.Spec.PublisherSpec.Type {
	case model.PublisherIpfs, model.PublisherEstuary:
		return nil
	default:
		return fmt.Errorf("invalid publisher type: %s", j.Spec.PublisherSpec.Type)
	}
}

func (publisher *IPFSPublisher) PublishResult(
	ctx context.Context,
	executionID string,
	j model.Job,
	resultPath string,
) (model.StorageSpec, error) {
	cid, err := publisher.IPFSClient.Put(ctx, resultPath)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return job.GetIPFSPublishedStorageSpec(executionID, j, model.StorageSourceIPFS, cid), nil
}

// Compile-time check that publisher implements the correct interface:
var _ publisher.Publisher = (*IPFSPublisher)(nil)
