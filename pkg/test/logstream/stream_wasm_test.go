//go:build unit || !integration

package logstream

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/logtest"
	"github.com/stretchr/testify/require"
	"github.com/vincent-petithory/dataurl"
)

func (s *LogStreamTestSuite) initStorage(source, target string) []model.StorageSpec {
	clients := s.stack.IPFSClients()
	s.Require().GreaterOrEqual(len(clients), 1, "No IPFS clients to upload to?")

	getStorage := scenario.StoredFile(source, target)
	storageList, stErr := getStorage(s.ctx, model.StorageSourceIPFS, clients...)
	s.Require().NoError(stErr)

	return storageList
}

func (s *LogStreamTestSuite) TestWasmOutputStream() {
	node := s.stack.Nodes[0]
	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineWasm)
	require.NoError(s.T(), err)

	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Duration(10)*time.Second)
	defer cancelFunc()

	job := model.Job{
		Metadata: model.Metadata{
			ID: "logstreamtest-wasm",
		},
		Spec: model.Spec{
			Engine: model.EngineWasm,
			Inputs: s.initStorage(
				"../../../testdata/wasm/logtest/inputs/cosmic_computer.txt", "/inputs/file.txt",
			),
			Wasm: model.JobSpecWasm{
				EntryPoint: "_start",
				Parameters: []string{"/inputs/file.txt"},
				EntryModule: model.StorageSpec{
					StorageSource: model.StorageSourceInline,
					URL:           dataurl.EncodeBytes(logtest.Program()),
				},
			},
		},
	}

	ready := make(chan struct{})

	go func() {
		cm := system.NewCleanupManager()
		s.T().Cleanup(func() { cm.Cleanup(context.Background()) })

		result := s.T().TempDir()

		execution := store.Execution{
			ID:  "test-execution",
			Job: job,
		}
		verifierMock, err := noop_verifier.NewNoopVerifierWithConfig(context.Background(), cm, noop_verifier.VerifierConfig{
			ExternalHooks: noop_verifier.VerifierExternalHooks{
				GetResultPath: func(ctx context.Context, executionID string, job model.Job) (string, error) {
					return result, nil
				},
			},
		})
		require.NoError(s.T(), err)

		ex, _ := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
		env, _ := executor.NewEnvironment(execution, ex.GetStorageProvider(s.ctx))
		env.Build(s.ctx, verifierMock)

		<-ready

		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		res, err := exec.Run(ctx, env)
		require.NoError(s.T(), err)
	}()

	ready <- struct{}{}

	r, err := waitForOutputStream(ctx, "test-execution", true, true, exec)
	require.NotNil(s.T(), r)

	sc := bufio.NewScanner(r)
	sc.Scan()
	require.Contains(s.T(), sc.Text(), "The Project Gutenberg EBook of The Cosmic Computer, by Henry Beam Piper")
}
