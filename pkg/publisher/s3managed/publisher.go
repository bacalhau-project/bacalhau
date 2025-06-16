package s3managed

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
)

type PublisherParams struct {
	NCLPublisherProvider ncl.PublisherProvider
}

// Compile-time check that publisher implements the correct interface:
var _ publisher.Publisher = (*Publisher)(nil)

type Publisher struct {
}

func (p Publisher) IsInstalled(ctx context.Context) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (p Publisher) ValidateJob(ctx context.Context, j models.Job) error {
	//TODO implement me
	panic("implement me")
}

func (p Publisher) PublishResult(ctx context.Context, execution *models.Execution, resultPath string) (models.SpecConfig, error) {
	//TODO implement me
	panic("implement me")
}

func NewPublisher(params PublisherParams) *Publisher {
	return &Publisher{}
}

// Register
