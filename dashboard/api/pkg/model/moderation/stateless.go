package moderation

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/localdb"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type statelessAutoVerifier struct {
	store localdb.LocalDB
	chain ResultsModerator
}

func NewStatelessModerator(store localdb.LocalDB, chain ResultsModerator) ResultsModerator {
	return &statelessAutoVerifier{store: store, chain: chain}
}

// Verify implements ResultsModerator
func (s *statelessAutoVerifier) Verify(ctx context.Context, req verifier.VerifierRequest) ([]verifier.VerifierResult, error) {
	job, err := s.store.GetJob(ctx, req.JobID)
	if err != nil {
		return nil, err
	}

	if len(job.Spec.Inputs) == 0 && job.Spec.Network.Disabled() {
		// Job is stateless, so there is no way it can have had access to anything sensitive.
		return generic.Map(req.Executions, func(e model.ExecutionState) verifier.VerifierResult {
			return verifier.VerifierResult{ExecutionID: e.ID(), Verified: true}
		}), nil
	}

	return s.chain.Verify(ctx, req)
}

var _ ResultsModerator = (*statelessAutoVerifier)(nil)
