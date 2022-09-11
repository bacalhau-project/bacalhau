package deterministic

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/filecoin-project/bacalhau/pkg/verifier/results"
	"golang.org/x/mod/sumdb/dirhash"
)

type DeterministicVerifier struct {
	stateResolver *job.StateResolver
	results       *results.Results
	encrypter     verifier.EncrypterFunction
	decrypter     verifier.DecrypterFunction
}

func NewDeterministicVerifier(
	ctx context.Context, cm *system.CleanupManager,

	resolver *job.StateResolver,
	encrypter verifier.EncrypterFunction,
	decrypter verifier.DecrypterFunction,
) (*DeterministicVerifier, error) {
	results, err := results.NewResults()
	if err != nil {
		return nil, err
	}
	return &DeterministicVerifier{
		stateResolver: resolver,
		results:       results,
		encrypter:     encrypter,
		decrypter:     decrypter,
	}, nil
}

func (deterministicVerifier *DeterministicVerifier) IsInstalled(ctx context.Context) (bool, error) {
	return true, nil
}

func (deterministicVerifier *DeterministicVerifier) GetShardResultPath(
	ctx context.Context,
	shard model.JobShard,
) (string, error) {
	return deterministicVerifier.results.EnsureShardResultsDir(shard.Job.ID, shard.Index)
}

func (deterministicVerifier *DeterministicVerifier) GetShardProposal(
	ctx context.Context,
	shard model.JobShard,
	shardResultPath string,
) ([]byte, error) {
	j, err := deterministicVerifier.stateResolver.GetJob(ctx, shard.Job.ID)
	if err != nil {
		return nil, err
	}
	if len(j.RequesterPublicKey) == 0 {
		return nil, fmt.Errorf("no RequesterPublicKey found in the job")
	}
	dirHash, err := dirhash.HashDir(shardResultPath, "results", dirhash.Hash1)
	if err != nil {
		return nil, err
	}
	encryptedHash, err := deterministicVerifier.encrypter(ctx, []byte(dirHash), j.RequesterPublicKey)
	if err != nil {
		return nil, err
	}
	return encryptedHash, nil
}

// each shard must have >= concurrency states
// and they must be either JobStateError or JobStateVerifying
func (deterministicVerifier *DeterministicVerifier) IsExecutionComplete(
	ctx context.Context,
	jobID string,
) (bool, error) {
	return deterministicVerifier.stateResolver.CheckShardStates(ctx, jobID, func(
		shardStates []model.JobShardState,
		concurrency int,
	) (bool, error) {
		return deterministicVerifier.results.CheckShardStates(shardStates, concurrency)
	})
}

func (deterministicVerifier *DeterministicVerifier) getHashGroups(
	ctx context.Context,
	jobData model.Job,
	shardStates []model.JobShardState,
) map[string][]*verifier.VerifierResult {
	// group the verifier results by their reported hash
	// then pick the largest group and verify all of those
	// caveats:
	//  * if there is only 1 group - there must be > 1 result
	//  * there cannot be a draw between the top 2 groups
	hashGroups := map[string][]*verifier.VerifierResult{}

	for _, shardState := range shardStates { //nolint:gocritic
		// we've already called IsExecutionComplete so will assume any shard state
		// that is not JobStateVerifying we can safely ignore
		if shardState.State != model.JobStateVerifying {
			continue
		}

		hash := ""

		if len(shardState.VerificationProposal) > 0 {
			decryptedHash, err := deterministicVerifier.decrypter(ctx, shardState.VerificationProposal)

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
			JobID:      jobData.ID,
			NodeID:     shardState.NodeID,
			ShardIndex: shardState.ShardIndex,
			Verified:   false,
		})
	}

	return hashGroups
}

func (deterministicVerifier *DeterministicVerifier) verifyShard(
	ctx context.Context,
	jobData model.Job,
	shardStates []model.JobShardState,
) ([]verifier.VerifierResult, error) {
	confidence := jobData.Deal.Confidence

	largestGroupHash := ""
	largestGroupSize := 0
	isVoidResult := false
	groupSizeCounts := map[int]int{}
	hashGroups := deterministicVerifier.getHashGroups(ctx, jobData, shardStates)

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

	allResults := []verifier.VerifierResult{}

	for _, verificationResultList := range hashGroups {
		for _, verificationResult := range verificationResultList {
			allResults = append(allResults, *verificationResult)
		}
	}

	return allResults, nil
}

func (deterministicVerifier *DeterministicVerifier) VerifyJob(
	ctx context.Context,
	jobID string,
) ([]verifier.VerifierResult, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/verifier/deterministic.VerifyJob")
	defer span.End()

	jobState, err := deterministicVerifier.stateResolver.GetJobState(ctx, jobID)
	if err != nil {
		return nil, err
	}

	jobData, err := deterministicVerifier.stateResolver.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	groupedShards := job.GroupShardStates(job.FlattenShardStates(jobState))
	allResults := []verifier.VerifierResult{}

	for _, shardStates := range groupedShards {
		shardResults, err := deterministicVerifier.verifyShard(ctx, jobData, shardStates)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, shardResults...)
	}

	return allResults, nil
}

// Compile-time check that deterministicVerifier implements the correct interface:
var _ verifier.Verifier = (*DeterministicVerifier)(nil)
