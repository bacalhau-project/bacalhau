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

type LogStreamServerOptions struct {
	Ctx            context.Context
	Host           host.Host
	ExecutionStore store.ExecutionStore
	Executors      executor.ExecutorProvider
}

func NewLogStreamServer(options LogStreamServerOptions) *LogStreamServer {
	svr := &LogStreamServer{
		ctx:            util.NewDetachedContext(options.Ctx),
		host:           options.Host,
		executionStore: options.ExecutionStore,
		executors:      options.Executors,
		Address:        findTCPAddress(options.Host),
	}
	svr.host.SetStreamHandler(LogsProcotolID, svr.Handle)
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

	log.Ctx(s.ctx).Debug().Msgf("Logserver read log header: %+v", request)

	execution, err := s.executionStore.GetExecution(s.ctx, request.ExecutionID)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("error retrieving execution: %s", request.ExecutionID)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Debug().Msgf("Logserver checking execution state: %+v", execution)

	if execution.State.IsTerminal() {
		// If the execution is already complete, possibly an error or possibly
		// just a really fast task, then we have to resort to reading the output
		// from the job. We will send the stdout/stderr that it did collect whilst
		// execution and will send stdout followed by stderr rather than the usual
		// interleaved dataframes.
		log.Ctx(s.ctx).Error().Msgf("execution was already terminated: %s", execution.ID)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Debug().Msgf("Logserver finding executor for: %+v", execution.Job.Spec.Engine)

	jobSpec := execution.Job.Spec
	e, err := s.executors.Get(s.ctx, jobSpec.Engine)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("failed to find executor for engine: %s", jobSpec.Engine)
		_ = stream.Reset()
		return
	}

	log.Ctx(s.ctx).Debug().Msgf("Logserver getting output stream")

	reader, err := e.GetOutputStream(s.ctx, execution.ID, request.WithHistory, request.Follow)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("failed to get output streams from job: %s", execution.Job.ID())
		_ = stream.Reset()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Ctx(s.ctx).Error().Msgf("source stream went away when sending logs to client")
			_ = stream.Reset()
		}
	}()

	_, err = io.Copy(stream, reader)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("problem reading from executor streams: %s", err)
	}

	_ = stream.Reset()
}
