package combo

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

// A fanoutPublisher is a publisher that will try multiple publishers in
// parallel and return the result from the first one to succeed. Other
// publishers will continue to run but their results and errors from the other
// publishers are also ignored. An error is only returned if all publishers fail
// to produce a result.
type fanoutPublisher struct {
	publishers []publisher.Publisher
}

func NewFanoutPublisher(publishers ...publisher.Publisher) publisher.Publisher {
	return &fanoutPublisher{publishers}
}

// fanout runs the passed method for all publishers in parallel. It immediately
// returns two channels from which the results can be read. Return values are
// written immediately to the value channel. A single error is written to the
// error channel only when all publishers have returned.
func fanout[T any, P any](ctx context.Context, publishers []P, method func(P) (T, error)) (chan T, chan error) {
	valueChannel := make(chan T, len(publishers))
	internalErrorChannel := make(chan error, len(publishers))
	externalErrorChannel := make(chan error, 1)

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(publishers))

	go func() {
		waitGroup.Wait()
		close(internalErrorChannel)
		var multi error
		for err := range internalErrorChannel {
			multi = multierr.Append(multi, err)
		}
		externalErrorChannel <- multi
		close(externalErrorChannel)
	}()

	runFunc := func(p P) {
		value, err := method(p)
		if err == nil {
			valueChannel <- value
			log.Ctx(ctx).Debug().Str("Publisher", fmt.Sprintf("%T", p)).Interface("Value", value).Send()
		} else {
			internalErrorChannel <- err
			log.Ctx(ctx).Error().Str("Publisher", fmt.Sprintf("%T", p)).Err(err).Send()
		}
		waitGroup.Done()
	}

	for _, publisher := range publishers {
		go runFunc(publisher)
	}

	return valueChannel, externalErrorChannel
}

// IsInstalled implements publisher.Publisher
func (f *fanoutPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx = log.Ctx(ctx).With().Str("Method", "IsInstalled").Logger().WithContext(ctx)

	valueChannel, errorChannel := fanout(ctx, f.publishers, func(p publisher.Publisher) (bool, error) {
		return p.IsInstalled(ctx)
	})

	// If we have a true result, return it right away. Else, wait for any other
	// publisher that might return a true result. If none do, the errorChannel
	// will close and if all publishers are actually fine err will just be nil.
	for {
		select {
		case installed := <-valueChannel:
			if installed {
				return installed, nil
			}
		case err := <-errorChannel:
			return false, err
		}
	}
}

// PublishShardResult implements publisher.Publisher
func (f *fanoutPublisher) PublishShardResult(
	ctx context.Context,
	shard model.JobShard,
	hostID string,
	shardResultPath string,
) (model.StorageSpec, error) {
	ctx = log.Ctx(ctx).With().Str("Method", "PublishShardResult").Logger().WithContext(ctx)

	valueChannel, errorChannel := fanout(ctx, f.publishers, func(p publisher.Publisher) (model.StorageSpec, error) {
		return p.PublishShardResult(ctx, shard, hostID, shardResultPath)
	})

	// Just return the first storage spec that we get
	select {
	case value := <-valueChannel:
		return value, nil
	case err := <-errorChannel:
		return model.StorageSpec{}, err
	}
}

var _ publisher.Publisher = (*fanoutPublisher)(nil)
