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
		storages   storage.StorageProvider
		executors  executor.ExecutorProvider
		verifiers  verifier.VerifierProvider
		publishers publisher.PublisherProvider
		bid        BidStrategyRequest
		shouldBid  bool
	}{
		{
			name:       "no-storage",
			storages:   dummy[model.StorageSourceType, storage.Storage](model.StorageSourceURLDownload),
			executors:  dummy[model.Engine, executor.Executor](model.EngineDocker),
			verifiers:  dummy[model.Verifier, verifier.Verifier](model.VerifierNoop),
			publishers: dummy[model.Publisher, publisher.Publisher](model.PublisherEstuary),
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
			storages:   dummy[model.StorageSourceType, storage.Storage](model.StorageSourceURLDownload),
			executors:  dummy[model.Engine, executor.Executor](model.EngineDocker),
			verifiers:  dummy[model.Verifier, verifier.Verifier](model.VerifierNoop),
			publishers: dummy[model.Publisher, publisher.Publisher](model.PublisherEstuary),
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
			storages:   dummy[model.StorageSourceType, storage.Storage](model.StorageSourceInline),
			executors:  dummy[model.Engine, executor.Executor](model.EngineWasm),
			verifiers:  dummy[model.Verifier, verifier.Verifier](model.VerifierDeterministic),
			publishers: dummy[model.Publisher, publisher.Publisher](model.PublisherEstuary),
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
			storages:   dummy[model.StorageSourceType, storage.Storage](model.StorageSourceURLDownload),
			executors:  dummy[model.Engine, executor.Executor](model.EngineDocker),
			verifiers:  dummy[model.Verifier, verifier.Verifier](model.VerifierNoop),
			publishers: dummy[model.Publisher, publisher.Publisher](model.PublisherEstuary),
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
			storages:   dummy[model.StorageSourceType, storage.Storage](model.StorageSourceFilecoin),
			executors:  dummy[model.Engine, executor.Executor](model.EngineDocker),
			verifiers:  dummy[model.Verifier, verifier.Verifier](model.VerifierNoop),
			publishers: dummy[model.Publisher, publisher.Publisher](model.PublisherFilecoin),
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
			storages:   dummy[model.StorageSourceType, storage.Storage](model.StorageSourceInline),
			executors:  dummy[model.Engine, executor.Executor](model.EngineWasm),
			verifiers:  dummy[model.Verifier, verifier.Verifier](model.VerifierDeterministic),
			publishers: dummy[model.Publisher, publisher.Publisher](model.PublisherIpfs),
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

type dummyProvider[Key model.ProviderKey, Value model.Providable] struct {
	key Key
}

// Get implements executor.ExecutorProvider
func (dummyProvider[Key, Value]) Get(context.Context, Key) (Value, error) {
	panic("unimplemented")
}

// Has implements executor.ExecutorProvider
func (d dummyProvider[Key, Value]) Has(_ context.Context, k Key) bool {
	return d.key == k
}

func dummy[Key model.ProviderKey, Value model.Providable](k Key) model.Provider[Key, Value] {
	return dummyProvider[Key, Value]{key: k}
}
