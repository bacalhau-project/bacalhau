package executor

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	EasterWASM          = "../../../testdata/wasm/"
	TestNodeConcurrency = 1
)

type ExecutorWASMExecutorSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestExecutorWasmExecutorSuite(t *testing.T) {
	suite.Run(t, new(ExecutorWASMExecutorSuite))
}

func (suite *ExecutorWASMExecutorSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
}

func (suite *ExecutorWASMExecutorSuite) TestWASMExecution() {
	ctx := context.Background()

	stack := testutils.NewDevStack(ctx, suite.T(), computenode.NewDefaultComputeNodeConfig())
	defer stack.Node.CleanupManager.Cleanup()

	wasmExecutor, err := stack.Node.Executors.GetExecutor(ctx, model.EngineWasm)
	require.NoError(suite.T(), err)

	cid, err := devstack.AddFileToNodes(ctx, EasterWASM, stack.IpfsStack.IPFSClients[:TestNodeConcurrency]...)
	require.NoError(suite.T(), err)

	inputStorageSpec := model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           cid,
	}

	isInstalled, err := wasmExecutor.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), isInstalled)

	hasStorage, err := wasmExecutor.HasStorageLocally(ctx, inputStorageSpec)
	require.NoError(suite.T(), err)
	require.True(suite.T(), hasStorage)

	job := &model.Job{
		ID:              "test-job",
		RequesterNodeID: "test-owner",
		ClientID:        "test-client",
		Spec: model.Spec{
			Engine: model.EngineDocker,
			Language: model.JobSpecLanguage{
				Language:        "wasm",
				LanguageVersion: "2.0",
				Deterministic:   true,
				Command:         "easter2022",
				ProgramPath:     "easter.wasm",
			},
			Contexts: []model.StorageSpec{inputStorageSpec},
		},
		Deal: model.Deal{
			Concurrency: TestNodeConcurrency,
		},
		CreatedAt: time.Now(),
	}

	shard := model.JobShard{
		Job:   job,
		Index: 0,
	}

	resultsDirectory, err := ioutil.TempDir("", "bacalhau-wasmExecutorTest")
	require.NoError(suite.T(), err)

	runnerOutput, err := wasmExecutor.RunShard(ctx, shard, resultsDirectory)
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), runnerOutput.ErrorMsg)

	const wasmProgramExitCode = 17
	require.Equal(suite.T(), wasmProgramExitCode, runnerOutput.ExitCode)
	require.NoError(suite.T(), err)
}
