package noop

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/results"
)

type VerifierHandlerIsInstalled func(ctx context.Context) (bool, error)
type VerifierHandlerGetResultPath func(ctx context.Context, executionID string, job model.Job) (string, error)
type VerifierHandlerGetProposal func(context.Context, model.Job, string, string) ([]byte, error)
type VerifierHandlerVerify func(context.Context, verifier.VerifierRequest) ([]verifier.VerifierResult, error)

type VerifierExternalHooks struct {
	IsInstalled   VerifierHandlerIsInstalled
	GetResultPath VerifierHandlerGetResultPath
	GetProposal   VerifierHandlerGetProposal
	Verify        VerifierHandlerVerify
}

type VerifierConfig struct {
	ExternalHooks VerifierExternalHooks
}

type NoopVerifier struct {
	results       *results.Results
	externalHooks VerifierExternalHooks
}

func NewNoopVerifier(
	_ context.Context, cm *system.CleanupManager,
) (*NoopVerifier, error) {
	results, err := results.NewResults()
	if err != nil {
		return nil, err
	}

	return &NoopVerifier{
		results: results,
	}, nil
}

func NewNoopVerifierWithConfig(ctx context.Context, cm *system.CleanupManager, config VerifierConfig) (*NoopVerifier, error) {
	v, err := NewNoopVerifier(ctx, cm)
	if err != nil {
		return nil, err
	}
	v.externalHooks = config.ExternalHooks
	return v, nil
}

func (noopVerifier *NoopVerifier) IsInstalled(ctx context.Context) (bool, error) {
	if noopVerifier.externalHooks.IsInstalled != nil {
		return noopVerifier.externalHooks.IsInstalled(ctx)
	}
	return true, nil
}

func (noopVerifier *NoopVerifier) GetResultPath(
	ctx context.Context,
	executionID string,
	job model.Job,
) (string, error) {
	if noopVerifier.externalHooks.GetResultPath != nil {
		return noopVerifier.externalHooks.GetResultPath(ctx, executionID, job)
	}
	return noopVerifier.results.EnsureResultsDir(executionID)
}

func (noopVerifier *NoopVerifier) GetProposal(
	ctx context.Context,
	job model.Job,
	executionID, resultsPath string,
) ([]byte, error) {
	if noopVerifier.externalHooks.GetProposal != nil {
		return noopVerifier.externalHooks.GetProposal(ctx, job, executionID, resultsPath)
	}
	return []byte{}, nil
}

func (noopVerifier *NoopVerifier) Verify(
	ctx context.Context,
	request verifier.VerifierRequest,
) ([]verifier.VerifierResult, error) {
	if noopVerifier.externalHooks.Verify != nil {
		return noopVerifier.externalHooks.Verify(ctx, request)
	}
	err := verifier.ValidateExecutions(request)
	if err != nil {
		return nil, err
	}

	verifierResults := make([]verifier.VerifierResult, 0, len(request.Executions))
	for _, execution := range request.Executions { //nolint:gocritic
		verifierResults = append(verifierResults, verifier.VerifierResult{
			ExecutionID: execution.ID(),
			Verified:    true,
		})
	}
	return verifierResults, nil
}

// Compile-time check that NoopVerifier implements the correct interface:
var _ verifier.Verifier = (*NoopVerifier)(nil)
