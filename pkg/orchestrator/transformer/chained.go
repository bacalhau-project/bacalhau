package transformer

import (
	"context"
)

// ChainedTransformer is a slice of Transformers that runs in sequence
type ChainedTransformer[T any] []GenericTransformer[T]

// Transform runs all transformers in sequence.
func (ct ChainedTransformer[T]) Transform(ctx context.Context, obj T) error {
	for _, t := range ct {
		if err := t.Transform(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}
