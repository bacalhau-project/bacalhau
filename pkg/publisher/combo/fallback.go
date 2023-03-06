package combo

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
)

type fallbackPublisher struct {
	publishers []publisher.Publisher
}

// NewFallbackPublisher returns a publisher.Publisher that will try multiple other
// Publishers in order until one succeeds.
//
// The Publishers are tried in the order specified in the call to
// NewFallbackPublisher. If and only if all the Publishers return an error
// result, the fallback publisher will also return an error result. Otherwise,
// it will return the result of the first Publisher to succeed and will swallow
// any errors. Subsequent Publishers will not be attempted after one succeeds.
func NewFallbackPublisher(publishers ...publisher.Publisher) publisher.Publisher {
	return &fallbackPublisher{
		publishers: publishers,
	}
}

// fallback accepts a slice of publisher objects and passes them to the supplied
// function in order, until one does not return an error value and then returns
// that result. If all publishers return an error value, a composite error is
// returned.
func fallback[T any, P any](ctx context.Context, publishers []P, method func(P) (T, error)) (T, error) {
	var anyErr error
	for _, publisher := range publishers {
		result, err := method(publisher)
		if err == nil {
			return result, nil
		} else {
			log.Ctx(ctx).Warn().Msgf("publisher %v returned an error: %s", publisher, err.Error())
			anyErr = multierr.Append(anyErr, err)
		}
	}

	var zeroResult T
	return zeroResult, anyErr
}

// IsInstalled implements publisher.Publisher
func (f *fallbackPublisher) IsInstalled(ctx context.Context) (bool, error) {
	return fallback(ctx, f.publishers, func(p publisher.Publisher) (bool, error) {
		return p.IsInstalled(ctx)
	})
}

// PublishResult implements publisher.Publisher
func (f *fallbackPublisher) PublishResult(
	ctx context.Context,
	job model.Job,
	hostID string,
	resultPath string,
) (model.StorageSpec, error) {
	return fallback(ctx, f.publishers, func(p publisher.Publisher) (model.StorageSpec, error) {
		return p.PublishResult(ctx, job, hostID, resultPath)
	})
}
