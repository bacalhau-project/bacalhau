//go:build unit || !integration

package logstream

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"
	"github.com/stretchr/testify/require"
	"github.com/vincent-petithory/dataurl"
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
		Spec: model.Spec{
			Engine: model.EngineWasm,
			Wasm: model.JobSpecWasm{
				EntryPoint: "_start",
				EntryModule: model.StorageSpec{
					StorageSource: model.StorageSourceInline,
					URL:           dataurl.EncodeBytes(cat.Program()),
				},
			},
		},
	}

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		_, err = exec.Run(ctx, job, "/tmp")
		require.NoError(s.T(), err)
	}()

	_, err = waitForOutputStream(ctx, job, true, exec)
	require.NotNil(s.T(), err)
}
