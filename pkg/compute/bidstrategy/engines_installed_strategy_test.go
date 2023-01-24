//go:build unit || !integration

package bidstrategy

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnginesInstalledStrategy(t *testing.T) {
	tests := []struct {
		name       string
		storages   dummyStorageProvider
		executors  dummyExecutorProvider
		verifiers  dummyVerifierProvider
		publishers dummyPublisherProvider
		bid        BidStrategyRequest
		shouldBid  bool
	}{
		{
			name:       "no-storage",
			storages:   map[model.StorageSourceType]struct{}{model.StorageSourceURLDownload: {}},
			executors:  map[model.Engine]struct{}{model.EngineDocker: {}},
			verifiers:  map[model.Verifier]struct{}{model.VerifierNoop: {}},
			publishers: map[model.Publisher]struct{}{model.PublisherEstuary: {}},
			bid: BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Inputs:    []model.StorageSpec{},
						Engine:    model.EngineDocker,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherEstuary,
					},
				},
			},
			shouldBid: true,
		},
		{
			name:       "invalid-storage",
			storages:   map[model.StorageSourceType]struct{}{model.StorageSourceURLDownload: {}},
			executors:  map[model.Engine]struct{}{model.EngineDocker: {}},
			verifiers:  map[model.Verifier]struct{}{model.VerifierNoop: {}},
			publishers: map[model.Publisher]struct{}{model.PublisherEstuary: {}},
			bid: BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Inputs: []model.StorageSpec{
							{
								StorageSource: model.StorageSourceIPFS,
							},
						},
						Engine:    model.EngineDocker,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherEstuary,
					},
				},
			},
			shouldBid: false,
		},
		{
			name:       "invalid-executor",
			storages:   map[model.StorageSourceType]struct{}{model.StorageSourceInline: {}},
			executors:  map[model.Engine]struct{}{model.EngineWasm: {}},
			verifiers:  map[model.Verifier]struct{}{model.VerifierDeterministic: {}},
			publishers: map[model.Publisher]struct{}{model.PublisherEstuary: {}},
			bid: BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Inputs: []model.StorageSpec{
							{
								StorageSource: model.StorageSourceInline,
							},
						},
						Engine:    model.EngineDocker,
						Verifier:  model.VerifierDeterministic,
						Publisher: model.PublisherEstuary,
					},
				},
			},
			shouldBid: false,
		},
		{
			name:       "invalid-verifier",
			storages:   map[model.StorageSourceType]struct{}{model.StorageSourceURLDownload: {}},
			executors:  map[model.Engine]struct{}{model.EngineDocker: {}},
			verifiers:  map[model.Verifier]struct{}{model.VerifierNoop: {}},
			publishers: map[model.Publisher]struct{}{model.PublisherEstuary: {}},
			bid: BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Inputs: []model.StorageSpec{
							{
								StorageSource: model.StorageSourceURLDownload,
							},
						},
						Engine:    model.EngineDocker,
						Verifier:  model.VerifierDeterministic,
						Publisher: model.PublisherEstuary,
					},
				},
			},
			shouldBid: false,
		},
		{
			name:       "invalid-publisher",
			storages:   map[model.StorageSourceType]struct{}{model.StorageSourceFilecoin: {}},
			executors:  map[model.Engine]struct{}{model.EngineDocker: {}},
			verifiers:  map[model.Verifier]struct{}{model.VerifierNoop: {}},
			publishers: map[model.Publisher]struct{}{model.PublisherFilecoin: {}},
			bid: BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Inputs: []model.StorageSpec{
							{
								StorageSource: model.StorageSourceFilecoin,
							},
						},
						Engine:    model.EngineDocker,
						Verifier:  model.VerifierNoop,
						Publisher: model.PublisherEstuary,
					},
				},
			},
			shouldBid: false,
		},
		{
			name:       "valid-request",
			storages:   map[model.StorageSourceType]struct{}{model.StorageSourceInline: {}},
			executors:  map[model.Engine]struct{}{model.EngineWasm: {}},
			verifiers:  map[model.Verifier]struct{}{model.VerifierDeterministic: {}},
			publishers: map[model.Publisher]struct{}{model.PublisherIpfs: {}},
			bid: BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{
						Inputs: []model.StorageSpec{
							{
								StorageSource: model.StorageSourceInline,
							},
						},
						Engine:    model.EngineWasm,
						Verifier:  model.VerifierDeterministic,
						Publisher: model.PublisherIpfs,
					},
				},
			},
			shouldBid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			subject := NewEnginesInstalledStrategy(EnginesInstalledStrategyParams{
				Storages:   test.storages,
				Executors:  test.executors,
				Verifiers:  test.verifiers,
				Publishers: test.publishers,
			})

			actual, err := subject.ShouldBid(context.Background(), test.bid)
			require.NoError(t, err)

			assert.Equal(t, test.shouldBid, actual.ShouldBid, actual.Reason)
		})
	}
}

var _ storage.StorageProvider = dummyStorageProvider{}

type dummyStorageProvider map[model.StorageSourceType]struct{}

func (d dummyStorageProvider) GetStorage(context.Context, model.StorageSourceType) (storage.Storage, error) {
	panic("not implemented")
}

func (d dummyStorageProvider) HasStorage(_ context.Context, sourceType model.StorageSourceType) bool {
	if _, ok := d[sourceType]; ok {
		return true
	}
	return false
}

var _ executor.ExecutorProvider = dummyExecutorProvider{}

type dummyExecutorProvider map[model.Engine]struct{}

func (d dummyExecutorProvider) AddExecutor(context.Context, model.Engine, executor.Executor) error {
	panic("not implemented")

}

func (d dummyExecutorProvider) GetExecutor(context.Context, model.Engine) (executor.Executor, error) {
	panic("not implemented")

}

func (d dummyExecutorProvider) HasExecutor(_ context.Context, engineType model.Engine) bool {
	if _, ok := d[engineType]; ok {
		return true
	}
	return false
}

var _ verifier.VerifierProvider = dummyVerifierProvider{}

type dummyVerifierProvider map[model.Verifier]struct{}

func (d dummyVerifierProvider) GetVerifier(context.Context, model.Verifier) (verifier.Verifier, error) {
	panic("not implemented")

}

func (d dummyVerifierProvider) HasVerifier(_ context.Context, job model.Verifier) bool {
	if _, ok := d[job]; ok {
		return true
	}
	return false
}

var _ publisher.PublisherProvider = dummyPublisherProvider{}

type dummyPublisherProvider map[model.Publisher]struct{}

func (d dummyPublisherProvider) GetPublisher(context.Context, model.Publisher) (publisher.Publisher, error) {
	panic("not implemented")

}

func (d dummyPublisherProvider) HasPublisher(_ context.Context, publisher model.Publisher) bool {
	if _, ok := d[publisher]; ok {
		return true
	}
	return false
}
