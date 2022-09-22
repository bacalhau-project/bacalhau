package filecoinlotus

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type FilecoinLotusPublisherConfig struct {
	ExecutablePath  string
	MinerAddress    string
	StoragePrice    string
	StorageDuration string
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
	if config.MinerAddress == "" {
		return nil, fmt.Errorf("MinerAddress is required")
	}
	if config.StoragePrice == "" {
		return nil, fmt.Errorf("StoragePrice is required")
	}
	if config.StorageDuration == "" {
		return nil, fmt.Errorf("StorageDuration is required")
	}
	return &FilecoinLotusPublisher{
		StateResolver: resolver,
		Config:        processedConfig,
	}, nil
}

func (lotusPublisher *FilecoinLotusPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/IsInstalled")
	defer span.End()

	_, err := lotusPublisher.runLotusCommand(ctx, []string{"exec", "lotus", "lotus", "--version"})
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
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/PublishShardResult")
	defer span.End()

	log.Debug().Msgf(
		"Uploading results folder to filecoin lotus: %s %s %s",
		hostID,
		shard,
		shardResultPath,
	)
	contentCid, err := lotusPublisher.importData(ctx, shardResultPath)
	if err != nil {
		return model.StorageSpec{}, err
	}
	dealCid, err := lotusPublisher.createDeal(ctx, contentCid)
	if err != nil {
		return model.StorageSpec{}, err
	}
	return model.StorageSpec{
		Name:   fmt.Sprintf("job-%s-shard-%d-host-%s", shard.Job.ID, shard.Index, hostID),
		Engine: model.StorageSourceFilecoin,
		Cid:    contentCid,
		Metadata: map[string]string{
			"deal_cid": dealCid,
		},
	}, nil
}

func (lotusPublisher *FilecoinLotusPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/ComposeResultReferences")
	defer span.End()

	system.AddJobIDFromBaggageToSpan(ctx, span)

	results := []model.StorageSpec{}
	shardResults, err := lotusPublisher.StateResolver.GetResults(ctx, jobID)
	if err != nil {
		return results, err
	}
	for _, shardResult := range shardResults {
		results = append(results, shardResult.Results)
	}
	return results, nil
}

//nolint:all
func (lotusPublisher *FilecoinLotusPublisher) tarResultsDir(ctx context.Context, resultsDir string) (string, error) {
	//nolint:ineffassign,staticcheck
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/tarResultsDir")
	defer span.End()

	tempDir, err := ioutil.TempDir("", "bacalhau-filecoin-lotus-test")
	if err != nil {
		return "", err
	}
	tempFile := fmt.Sprintf("%s/results.tar", tempDir)
	_, err = system.UnsafeForUserCodeRunCommand("tar", []string{
		"-cvf",
		tempFile,
		resultsDir,
	})
	if err != nil {
		return "", err
	}
	return tempFile, nil
}

func (lotusPublisher *FilecoinLotusPublisher) importData(ctx context.Context, filePath string) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/importData")
	defer span.End()

	r, err := lotusPublisher.runLotusCommand(ctx, []string{"exec", "lotus", "lotus", "client", "import", filePath})
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.TrimSpace(r.STDOUT), " ")
	return parts[len(parts)-1], nil
}

func (lotusPublisher *FilecoinLotusPublisher) ListDeals(ctx context.Context) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/ListDeals")
	defer span.End()

	r, err := lotusPublisher.runLotusCommand(ctx, []string{"exec", "lotus", "lotus", "client", "list-deals", "-v"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(r.STDOUT), nil
}

func (lotusPublisher *FilecoinLotusPublisher) createDeal(ctx context.Context, contentCid string) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/createDeal")
	defer span.End()

	r, err := lotusPublisher.runLotusCommand(ctx, []string{
		"exec", "lotus", "lotus", "client", "deal",
		contentCid,
		lotusPublisher.Config.MinerAddress,
		lotusPublisher.Config.StoragePrice,
		lotusPublisher.Config.StorageDuration,
	})
	if err != nil {
		return "", err
	}
	dealCid := ""
	for _, line := range strings.Split(strings.TrimSpace(r.STDOUT), "\n") {
		dealCid = line
	}
	if dealCid == "" {
		return "", fmt.Errorf("no deal cid found in output")
	}
	return dealCid, nil
}

func (lotusPublisher *FilecoinLotusPublisher) runLotusCommand(ctx context.Context, args []string) (*model.RunCommandResult, error) {
	//nolint:ineffassign,staticcheck
	_, span := system.GetTracer().Start(ctx, "pkg/publisher/filecoin_lotus/runLotusCommand")
	defer span.End()

	return system.UnsafeForUserCodeRunCommand(lotusPublisher.Config.ExecutablePath, args)
}

func processConfig(config FilecoinLotusPublisherConfig) (FilecoinLotusPublisherConfig, error) {
	if config.ExecutablePath == "" {
		r, err := system.UnsafeForUserCodeRunCommand("which", []string{"lotus"})
		if err != nil {
			return config, err
		}
		config.ExecutablePath = r.STDOUT
	}
	return config, nil
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*FilecoinLotusPublisher)(nil)
