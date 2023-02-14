package ipfs

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
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

func (publisher *IPFSPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	cid, err := publisher.IPFSClient.Put(ctx, shardResultPath)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return job.GetPublishedStorageSpec(shard, model.StorageSourceIPFS, hostID, cid), nil
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*IPFSPublisher)(nil)
