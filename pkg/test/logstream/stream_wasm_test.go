//go:build unit || !integration

package logstream

import (
	"context"
	"os"
	"time"

	"github.com/stretchr/testify/require"

	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

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
		Spec: testutils.MakeSpecWithOpts(s.T(),
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(
					model.StorageSpec{
						StorageSource: model.StorageSourceInline,
						URL:           dataurl.EncodeBytes(cat.Program()),
					},
				).WithEntrypoint("_start").
					Build(),
			),
		),
	}

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		dir, _ := os.MkdirTemp("", "test")
		_, err = exec.Run(ctx, "test-execution", job, dir)
		require.NoError(s.T(), err)
	}()

	_, err = waitForOutputStream(ctx, "test-execution", true, true, exec)
	require.NotNil(s.T(), err)
}
