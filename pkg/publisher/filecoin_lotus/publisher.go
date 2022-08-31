package filecoin_lotus

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type FilecoinLotusPublisherConfig struct {
	ExecutablePath string
}

type FilecoinLotusPublisher struct {
	StateResolver *job.StateResolver
	Config        FilecoinLotusPublisherConfig
}

func NewFilecoinLotusPublisher(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	config FilecoinLotusPublisherConfig,
) (*FilecoinLotusPublisher, error) {
	processedConfig, err := processConfig(config)
	if err != nil {
		return nil, err
	}
	return &FilecoinLotusPublisher{
		StateResolver: resolver,
		Config:        processedConfig,
	}, nil
}

func (lotusPublisher *FilecoinLotusPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	_, err := lotusPublisher.runLotusCommand(ctx, []string{"version"})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (lotusPublisher *FilecoinLotusPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	ctx, span := newSpan(ctx, "PublishShardResult")
	defer span.End()
	log.Debug().Msgf(
		"Uploading results folder to filecoin lotus: %s %s %s",
		hostID,
		shard,
		shardResultPath,
	)
	return model.StorageSpec{
		Name:   fmt.Sprintf("job-%s-shard-%d-host-%s", shard.Job.ID, shard.Index, hostID),
		Engine: model.StorageSourceFilecoin,
		Cid:    "123",
	}, nil
}

func (lotusPublisher *FilecoinLotusPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	results := []model.StorageSpec{}
	ctx, span := newSpan(ctx, "ComposeResultSet")
	defer span.End()
	shardResults, err := lotusPublisher.StateResolver.GetResults(ctx, jobID)
	if err != nil {
		return results, err
	}
	for _, shardResult := range shardResults {
		results = append(results, shardResult.Results)
	}
	return results, nil
}

func (lotusPublisher *FilecoinLotusPublisher) runLotusCommand(ctx context.Context, args []string) (string, error) {
	ctx, span := newSpan(ctx, "runLotusCommand")
	defer span.End()
	return system.RunCommandGetResults(lotusPublisher.Config.ExecutablePath, args)
}

func processConfig(config FilecoinLotusPublisherConfig) (FilecoinLotusPublisherConfig, error) {
	if config.ExecutablePath == "" {
		result, err := system.RunCommandGetResults("which", []string{"lotus"})
		if err != nil {
			return config, err
		}
		config.ExecutablePath = result
	}
	return config, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/filecoin_lotus", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*FilecoinLotusPublisher)(nil)
