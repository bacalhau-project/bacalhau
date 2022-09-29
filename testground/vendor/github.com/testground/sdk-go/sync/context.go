package sync

import (
	"context"

	"github.com/testground/sdk-go/runtime"
)

type runparamsCtxKey struct{}

var runparams = runparamsCtxKey{}

// WithRunParams returns a context that embeds the supplied RunParams,
// such that it can be passed to a GenericClient.
func WithRunParams(ctx context.Context, rp *runtime.RunParams) context.Context {
	return context.WithValue(ctx, runparams, rp)
}

// GetRunParams extracts the RunParams from a context, previously set by calling
// WithRunParams.
func GetRunParams(ctx context.Context) *runtime.RunParams {
	v := ctx.Value(runparams)
	if v == nil {
		return nil
	}
	return v.(*runtime.RunParams)
}
