package ipfs

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type IPFSPublisher struct {
	IPFSClient    *ipfs.Client
	StateResolver *job.StateResolver
}

func NewIPFSPublisher(
	ctx context.Context,
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	ipfsAPIAddr string,
) (*IPFSPublisher, error) {
	cl, err := ipfs.NewClient(ipfsAPIAddr)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("IPFS publisher initialized for node: %s", ipfsAPIAddr)
	return &IPFSPublisher{
		IPFSClient:    cl,
		StateResolver: resolver,
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
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/ipfs.PublishShardResult")
	defer span.End()

	fmt.Printf(" --------------------------------------\n")
	fmt.Printf(" --------------------------------------\n")
	fmt.Printf(" --------------------------------------\n")
	fmt.Printf(" --------------------------------------\n")
	fmt.Printf(" --------------------------------------\n")
	log.Debug().Msgf(
		"Uploading results folder to ipfs: %s %s %s",
		hostID,
		shard,
		shardResultPath,
	)
	cid, err := publisher.IPFSClient.Put(ctx, shardResultPath)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return model.StorageSpec{
		Name:   fmt.Sprintf("job-%s-shard-%d-host-%s", shard.Job.ID, shard.Index, hostID),
		Engine: model.StorageSourceIPFS,
		Cid:    cid,
	}, nil
}

func (publisher *IPFSPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/ipfs.ComposeResultReferences")
	defer span.End()

	system.AddJobIDFromBaggageToSpan(ctx, span)

	results := []model.StorageSpec{}
	shardResults, err := publisher.StateResolver.GetResults(ctx, jobID)
	if err != nil {
		return results, err
	}
	for _, shardResult := range shardResults {
		results = append(results, shardResult.Results)
	}
	return results, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*IPFSPublisher)(nil)
