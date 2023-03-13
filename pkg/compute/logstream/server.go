package logstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog/log"
)

const (
	LogsProcotolID = "/bacalhau/compute/logs/1.0.0"
)

func NewLogStreamServer(
	ctx context.Context,
	host host.Host,
	executionStore store.ExecutionStore,
	executors executor.ExecutorProvider) *LogStreamServer {
	svr := &LogStreamServer{
		ctx:            util.NewDetachedContext(ctx),
		host:           host,
		executionStore: executionStore,
		executors:      executors,
	}
	svr.Address = findTCPAddress(host)

	host.SetStreamHandler(LogsProcotolID, svr.Handle)
	return svr
}

func findTCPAddress(host host.Host) string {
	peerID := host.ID().Pretty()
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", peerID))

	for _, addr := range host.Addrs() {
		for _, protocol := range addr.Protocols() {
			if protocol.Name == "tcp" {
				return addr.Encapsulate(hostAddr).String()
			}
		}
	}

	// If we can't find TCP, then we'll go with the first record
	addr := host.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func (s *LogStreamServer) Handle(stream network.Stream) {
	log.Ctx(s.ctx).Info().Msg("Handling new logging request")

	defer stream.Close()

	request := LogStreamRequest{}
	err := json.NewDecoder(stream).Decode(&request)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("error decoding %s: %s", reflect.TypeOf(request), err)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Info().Msgf("Logserver read log header: %+v", request)

	execution, err := s.executionStore.GetExecution(s.ctx, request.ExecutionID)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("error retrieving execution: %s", request.ExecutionID)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Info().Msgf("Logserver checking execution state: %+v", execution)

	if execution.State.IsTerminal() {
		log.Ctx(s.ctx).Error().Msgf("execution is already complete: %s", request.ExecutionID)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Info().Msgf("Logserver finding executor for: %+v", execution.Job.Spec.Engine)

	jobSpec := execution.Job.Spec
	e, err := s.executors.Get(s.ctx, jobSpec.Engine)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("failed to find executor for engine: %s", jobSpec.Engine)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Info().Msgf("Logserver getting output stream")

	reader, err := e.GetOutputStream(s.ctx, execution.Job)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("failed to get output streams from job: %s", execution.Job.ID())
		_ = stream.Reset()
		return
	}

	// While we can read, and don't get an EOF, keep writing to the stream.
	buffer := make([]byte, 65535) //nolint:gomnd
	for {
		log.Ctx(s.ctx).Info().Msgf("Logserver waiting for read ....")
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Ctx(s.ctx).Error().Msgf("problem reading from executor streams: %s", err)
			break
		}

		_, err = stream.Write(buffer[:n])
		if err != nil {
			log.Ctx(s.ctx).Error().Msgf("problem writing to stream: %s", err)
			break
		}
	}
	_ = stream.Reset()
}
