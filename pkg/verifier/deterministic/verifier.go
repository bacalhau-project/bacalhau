package deterministic

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/results"
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

	cm.RegisterCallback(func() error {
		if err := results.Close(); err != nil {
			return fmt.Errorf("unable to remove results folder: %w", err)
		}
		return nil
	})
	return &DeterministicVerifier{
		results:   results,
		encrypter: encrypter,
		decrypter: decrypter,
	}, nil
}

func (deterministicVerifier *DeterministicVerifier) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (deterministicVerifier *DeterministicVerifier) GetShardResultPath(
	_ context.Context,
	shard model.JobShard,
) (string, error) {
	return deterministicVerifier.results.EnsureShardResultsDir(shard.Job.Metadata.ID, shard.Index)
}

func (deterministicVerifier *DeterministicVerifier) GetShardProposal(
	ctx context.Context,
	shard model.JobShard,
	shardResultPath string,
) ([]byte, error) {
	if len(shard.Job.Metadata.Requester.RequesterPublicKey) == 0 {
		return nil, fmt.Errorf("no RequesterPublicKey found in the job")
	}
	dirHash, err := dirhash.HashDir(shardResultPath, "results", dirhash.Hash1)
	if err != nil {
		return nil, err
	}
	encryptedHash, err := deterministicVerifier.encrypter(ctx, []byte(dirHash), shard.Job.Metadata.Requester.RequesterPublicKey)
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
			Execution: executionState,
			Verified:  false,
		})
	}

	return hashGroups
}

func (deterministicVerifier *DeterministicVerifier) VerifyShard(
	ctx context.Context,
	shard model.JobShard,
	executionStates []model.ExecutionState,
) ([]verifier.VerifierResult, error) {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/verifier.DeterministicVerifier.VerifyShard")
	defer span.End()

	err := verifier.ValidateExecutions(shard, executionStates)
	if err != nil {
		return nil, err
	}
	confidence := shard.Job.Spec.Deal.Confidence

	largestGroupHash := ""
	largestGroupSize := 0
	isVoidResult := false
	groupSizeCounts := map[int]int{}
	hashGroups := deterministicVerifier.getHashGroups(ctx, executionStates)

	for hash, group := range hashGroups {
		if len(group) > largestGroupSize {
			largestGroupSize = len(group)
			largestGroupHash = hash
		}
		groupSizeCounts[len(group)]++
	}

	// this means there is a draw for the largest group size
	if groupSizeCounts[largestGroupSize] > 1 {
		isVoidResult = true
	}

	// this means there is only a single result
	if len(hashGroups) == 1 && largestGroupSize == 1 {
		isVoidResult = true
	}

	// this means that the winning group size does not
	// meet the confidence threshold
	if confidence > 0 && largestGroupSize < confidence {
		isVoidResult = true
	}

	// the winning hash must not be empty string
	if largestGroupHash == "" {
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
