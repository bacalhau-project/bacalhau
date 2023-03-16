//go:build unit || !integration

package logstream

import (
	"encoding/json"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	logstream_util "github.com/bacalhau-project/bacalhau/pkg/util/logstream"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/stretchr/testify/require"

	ma "github.com/multiformats/go-multiaddr"
)

func (s *LogStreamTestSuite) TestStreamAddress() {
	node := s.stack.Nodes[0]

	job := newDockerJob("address-test", model.JobSpecDocker{
		Image:      "bash",
		Entrypoint: []string{"bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done"},
	})

	execution := newTestExecution("test-execution", job)

	err := node.RequesterNode.JobStore.CreateJob(s.ctx, job)
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		require.NoError(s.T(), err)

		_, err = exec.Run(s.ctx, job, "/tmp")
		require.NoError(s.T(), err, err.Error())
	}()

	// Wait for the docker container to be running so we know it'll be there when
	// the logstream requests it
	reader, err := waitForOutputStream(s.ctx, job, true, exec)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	node.ComputeNode.ExecutionStore.CreateExecution(s.ctx, execution)
	err = node.RequesterNode.JobStore.CreateExecution(s.ctx, model.ExecutionState{
		State:            model.ExecutionStateBidAccepted,
		JobID:            job.ID(),
		ComputeReference: execution.ID,
		NodeID:           node.Host.ID().Pretty(),
	})
	require.NoError(s.T(), err)

	logRequest := requester.ReadLogsRequest{JobID: job.ID(), ExecutionID: execution.ID}
	response, err := node.RequesterNode.Endpoint.ReadLogs(s.ctx, logRequest)
	require.NoError(s.T(), err)

	host, err := libp2p.New([]libp2p.Option{libp2p.DisableRelay()}...)
	require.NoError(s.T(), err)

	maddr, err := ma.NewMultiaddr(response.Address)
	if err != nil {
		return
	}
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return
	}

	addresses := host.Peerstore().Addrs(info.ID)
	if len(addresses) == 0 {
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.TempAddrTTL)
	}

	stream, err := host.NewStream(s.ctx, info.ID, "/bacalhau/compute/logs/1.0.0")
	if err != nil {
		return
	}
	defer stream.Close()

	lsReq := logstream.LogStreamRequest{
		JobID:       job.ID(),
		ExecutionID: execution.ID,
		WithHistory: true,
	}

	err = json.NewEncoder(stream).Encode(lsReq)
	require.NoError(s.T(), err)

	frame, err := logstream_util.NewDataFrameFromReader(stream)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), frame)

	require.Equal(s.T(), string(frame.Data), "logstreamoutput\n")
}
