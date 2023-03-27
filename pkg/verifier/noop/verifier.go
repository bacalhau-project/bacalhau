package noop

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/results"
)

type VerifierHandlerIsInstalled func(ctx context.Context) (bool, error)
type VerifierHandlerGetResultPath func(ctx context.Context, job model.Job) (string, error)
type VerifierHandlerGetProposal func(context.Context, model.Job, string) ([]byte, error)
type VerifierHandlerVerify func(context.Context, model.Job, []model.ExecutionState) ([]verifier.VerifierResult, error)

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

	cm.RegisterCallback(func() error {
		if err := results.Close(); err != nil {
			return fmt.Errorf("unable to remove results folder: %w", err)
		}
		return nil
	})
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
	job model.Job,
	executorID string,
) (string, error) {
	if noopVerifier.externalHooks.GetResultPath != nil {
		return noopVerifier.externalHooks.GetResultPath(ctx, job)
	}
	return noopVerifier.results.EnsureResultsDir(job.ID(), executorID)
}

func (noopVerifier *NoopVerifier) GetProposal(
	ctx context.Context,
	job model.Job,
	s string,
) ([]byte, error) {
	if noopVerifier.externalHooks.GetProposal != nil {
		return noopVerifier.externalHooks.GetProposal(ctx, job, s)
	}
	return []byte{}, nil
}

func (noopVerifier *NoopVerifier) Verify(
	ctx context.Context,
	job model.Job,
	executionStates []model.ExecutionState,
) ([]verifier.VerifierResult, error) {
	if noopVerifier.externalHooks.Verify != nil {
		return noopVerifier.externalHooks.Verify(ctx, job, executionStates)
	}
	err := verifier.ValidateExecutions(job, executionStates)
	if err != nil {
		return nil, err
	}

	var verifierResults []verifier.VerifierResult
	for _, execution := range executionStates { //nolint:gocritic
		verifierResults = append(verifierResults, verifier.VerifierResult{
			Execution: execution,
			Verified:  true,
		})
	}
	return verifierResults, nil
}

// Compile-time check that NoopVerifier implements the correct interface:
var _ verifier.Verifier = (*NoopVerifier)(nil)
