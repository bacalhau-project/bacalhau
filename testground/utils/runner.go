package utils

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/testground/sdk-go/runtime"
)

// Run a test scenario using docker engine.
func RunDockerTest(
	runenv *runtime.RunEnv,
	ctx context.Context,
	testCase scenario.TestCase,
	node *node.Node,
	concurrency int,
) error {
	runenv.RecordMessage("Running scenario %v", testCase.Name)

	// TODO: the test uploads the input file to all nodes. The test should cover the cases where the input
	//  file is only available is some nodes in the network.
	inputStorageList, err := testCase.SetupStorage(ctx, model.StorageSourceIPFS, node.IPFSClient)
	if err != nil {
		return err
	}

	var j = &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker:    testCase.GetJobSpec(),
		Inputs:    inputStorageList,
		Outputs:   testCase.Outputs,
	}

	j.Deal = model.Deal{
		Concurrency: concurrency,
	}

	apiURI := node.APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiURI)
	submittedJob, err := apiClient.Submit(ctx, j, nil)
	runenv.RecordMessage("Submitted %v", testCase.Name)

	if err != nil {
		return err
	}

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		concurrency,
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: concurrency,
		}),
	)
	if err != nil {
		return err
	}

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	if err != nil {
		return err
	}

	// now we check the actual results produced by the ipfs verifier
	for i := range shards {
		outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-testground")
		if err != nil {
			return err
		}

		outputPath := filepath.Join(outputDir, shards[i].PublishedResult.CID)
		err = node.IPFSClient.Get(ctx, shards[i].PublishedResult.CID, outputPath)
		if err != nil {
			return err
		}

		err = testCase.ResultsChecker(outputPath)
		if err != nil {
			return err
		}
	}
	return nil
}
