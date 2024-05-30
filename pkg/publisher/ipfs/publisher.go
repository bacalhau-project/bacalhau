package ipfs

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

type IPFSPublisher struct {
	IPFSClient ipfs.Client
}

func NewIPFSPublisher(
	ctx context.Context,
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

func (publisher *IPFSPublisher) ValidateJob(ctx context.Context, j models.Job) error {
	switch j.Task().Publisher.Type {
	case models.PublisherIPFS:
		return nil
	default:
		return fmt.Errorf("invalid publisher type: %s", j.Task().Publisher.Type)
	}
}

func (publisher *IPFSPublisher) PublishResult(
	ctx context.Context,
	execution *models.Execution,
	resultPath string,
) (models.SpecConfig, error) {
	cid, err := publisher.IPFSClient.Put(ctx, resultPath)
	if err != nil {
		return models.SpecConfig{}, err
	}
	return models.SpecConfig{
		Type: models.StorageSourceIPFS,
		Params: map[string]interface{}{
			"CID": cid,
		},
	}, nil
}

// Compile-time check that publisher implements the correct interface:
var _ publisher.Publisher = (*IPFSPublisher)(nil)
