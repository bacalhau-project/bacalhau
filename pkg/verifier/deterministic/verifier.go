package deterministic

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
	"github.com/bacalhau-project/bacalhau/pkg/verifier/results"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/sumdb/dirhash"
)

type DeterministicVerifier struct {
	results   *results.Results
	encrypter verifier.EncrypterFunction
	decrypter verifier.DecrypterFunction
}

func NewDeterministicVerifier(
	_ context.Context, cm *system.CleanupManager,
	encrypter verifier.EncrypterFunction,
	decrypter verifier.DecrypterFunction,
) (*DeterministicVerifier, error) {
	results, err := results.NewResults()
	if err != nil {
		return nil, err
	}

	return &DeterministicVerifier{
		results:   results,
		encrypter: encrypter,
		decrypter: decrypter,
	}, nil
}

func (deterministicVerifier *DeterministicVerifier) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (deterministicVerifier *DeterministicVerifier) GetResultPath(
	_ context.Context,
	executionID string,
	job model.Job,
) (string, error) {
	return deterministicVerifier.results.EnsureResultsDir(executionID)
}

func (deterministicVerifier *DeterministicVerifier) GetProposal(
	ctx context.Context,
	job model.Job,
	executionID string,
	resultPath string,
) ([]byte, error) {
	if len(job.Metadata.Requester.RequesterPublicKey) == 0 {
		return nil, fmt.Errorf("no RequesterPublicKey found in the job")
	}
	dirHash, err := dirhash.HashDir(resultPath, "results", dirhash.Hash1)
	if err != nil {
		return nil, err
	}
	encryptedHash, err := deterministicVerifier.encrypter(ctx, []byte(dirHash), job.Metadata.Requester.RequesterPublicKey)
	if err != nil {
		return nil, err
	}
	return encryptedHash, nil
}

func (deterministicVerifier *DeterministicVerifier) getHashGroups(
	ctx context.Context,
	executionStates []model.ExecutionState,
) map[string][]*verifier.VerifierResult {
	// group the verifier results by their reported hash
	// then pick the largest group and verify all of those
	// caveats:
	//  * if there is only 1 group - there must be > 1 result
	//  * there cannot be a draw between the top 2 groups
	hashGroups := map[string][]*verifier.VerifierResult{}

	for _, executionState := range executionStates { //nolint:gocritic
		hash := ""

		if len(executionState.VerificationProposal) > 0 {
			decryptedHash, err := deterministicVerifier.decrypter(ctx, executionState.VerificationProposal)

			// if there is an error decrypting let's not fail the verification job
			// but just leave the proposed hash at empty string (which won't pass actual verification)
			// this means we can "complete" the verification process by deciding that anyone
			// who couldn't submit a correctly encrypted hash will result in an empty hash
			// rather than a decryption error
			if err == nil {
				hash = string(decryptedHash)
			}
		}

		existingArray, ok := hashGroups[hash]
		if !ok {
			existingArray = []*verifier.VerifierResult{}
		}
		hashGroups[hash] = append(existingArray, &verifier.VerifierResult{
			ExecutionID: executionState.ID(),
			Verified:    false,
		})
	}

	return hashGroups
}

func (deterministicVerifier *DeterministicVerifier) Verify(
	ctx context.Context,
	request verifier.VerifierRequest,
) ([]verifier.VerifierResult, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.DeterministicVerifier.Verify")
	defer span.End()

	err := verifier.ValidateExecutions(request)
	if err != nil {
		return nil, err
	}

	largestGroupHash := ""
	largestGroupSize := 0
	totalExecutions := 0
	isVoidResult := false
	groupSizeCounts := map[int]int{}
	hashGroups := deterministicVerifier.getHashGroups(ctx, request.Executions)

	for hash, group := range hashGroups {
		if len(group) > largestGroupSize {
			largestGroupSize = len(group)
			largestGroupHash = hash
		}
		groupSizeCounts[len(group)]++
		totalExecutions += len(group)
	}

	// this means there is a draw for the largest group size
	if groupSizeCounts[largestGroupSize] > 1 {
		log.Ctx(ctx).Debug().Str("Reason", "Draw for largest group size").Msg("Failing verification")
		isVoidResult = true
	}

	// this means there is only a single result
	if len(hashGroups) == 1 && largestGroupSize == 1 {
		log.Ctx(ctx).Debug().Str("Reason", "Only a single result").Msg("Failing verification")
		isVoidResult = true
	}

	// this means that the winning group size does not
	// meet the confidence threshold
	confidence := request.Deal.Confidence
	if confidence > 0 && largestGroupSize < confidence {
		log.Ctx(ctx).Debug().Str("Reason", "Largest group size below confidence").Msg("Failing verification")
		isVoidResult = true
	}

	// the winning hash must not be empty string
	if largestGroupHash == "" {
		log.Ctx(ctx).Debug().Str("Reason", "Hash is empty string").Msg("Failing verification")
		isVoidResult = true
	}

	if !isVoidResult {
		for _, passedVerificationResult := range hashGroups[largestGroupHash] {
			passedVerificationResult.Verified = true
		}
	}

	var allResults []verifier.VerifierResult

	for _, verificationResultList := range hashGroups {
		for _, verificationResult := range verificationResultList {
			allResults = append(allResults, *verificationResult)
		}
	}

	return allResults, nil
}

// Compile-time check that deterministicVerifier implements the correct interface:
var _ verifier.Verifier = (*DeterministicVerifier)(nil)
