package filecoinlotus

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type EstuaryPublisherConfig struct {
	APIKey string
}

type EstuaryPublisher struct {
	StateResolver *job.StateResolver
	Config        EstuaryPublisherConfig
}

func NewEstuaryPublisher(
	cm *system.CleanupManager,
	resolver *job.StateResolver,
	config EstuaryPublisherConfig,
) (*EstuaryPublisher, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}
	return &EstuaryPublisher{
		StateResolver: resolver,
		Config:        config,
	}, nil
}

func (estuaryPublisher *EstuaryPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	// client := &http.Client{}
	// req, err := http.NewRequest("GET", "https://icanhazdadjoke.com/", nil)
	// if err != nil {
	// 	fmt.Print(err.Error())
	// }
	// req.Header.Add("Accept", "application/json")
	// req.Header.Add("Content-Type", "application/json")

	// _, err := estuaryPublisher.runLotusCommand(ctx, []string{"version"})
	// if err != nil {
	// 	return false, err
	// }
	return true, nil
}

func (estuaryPublisher *EstuaryPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	ctx, span := newSpan(ctx, "PublishShardResult")
	defer span.End()
	log.Debug().Msgf(
		"Uploading results folder to estuary: %s %s %s",
		hostID,
		shard,
		shardResultPath,
	)
	// tarFile, err := estuaryPublisher.tarResultsDir(ctx, shardResultPath)
	// if err != nil {
	// 	return model.StorageSpec{}, err
	// }
	// contentCid, err := estuaryPublisher.importData(ctx, tarFile)
	// if err != nil {
	// 	return model.StorageSpec{}, err
	// }
	// dealCid, err := estuaryPublisher.createDeal(ctx, contentCid)
	// if err != nil {
	// 	return model.StorageSpec{}, err
	// }
	return model.StorageSpec{}, nil
}

func (estuaryPublisher *EstuaryPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]model.StorageSpec, error) {
	results := []model.StorageSpec{}
	ctx, span := newSpan(ctx, "ComposeResultSet")
	defer span.End()
	return results, nil
}

func (estuaryPublisher *EstuaryPublisher) tarResultsDir(ctx context.Context, resultsDir string) (string, error) {
	_, span := newSpan(ctx, "tarResultsDir")
	defer span.End()
	tempDir, err := ioutil.TempDir("", "bacalhau-filecoin-lotus-test")
	if err != nil {
		return "", err
	}
	tempFile := fmt.Sprintf("%s/results.tar", tempDir)
	_, err = system.RunCommandGetResults("tar", []string{
		"-cvf",
		tempFile,
		resultsDir,
	})
	if err != nil {
		return "", err
	}
	return tempFile, nil
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/estuary", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*EstuaryPublisher)(nil)
