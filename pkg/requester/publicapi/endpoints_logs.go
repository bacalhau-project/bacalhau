package publicapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/logstream"
	"github.com/gorilla/websocket"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	ma "github.com/multiformats/go-multiaddr"
)

type logRequest = SignedRequest[model.LogsPayload] //nolint:unused // Swagger wants this

type LogStreamRequest struct {
	JobID       string
	ExecutionID string
	WithHistory bool
}

type Msg struct {
	Tag  uint8
	Data string
}

// logs godoc
//
//	@ID						pkg/requester/publicapi/logs
//	@Summary				Displays the logs for a current job/execution
//	@Description.markdown	endpoints_log
//	@Tags					Job
//	@Accept					json
//	@Produce				json
//	@Param					logRequest  	body		logRequest	true	" "
//	@Success				200				{object}	string
//	@Failure				400				{object}	string
//	@Failure				401				{object}	string
//	@Failure				403				{object}	string
//	@Failure				500				{object}	string
//	@Router					/requester/logs [post]
//
//nolint:funlen,gocyclo
func (s *RequesterAPIServer) logs(res http.ResponseWriter, req *http.Request) {
	var upgrader = websocket.Upgrader{}
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("failed to upgrade websocket connection: %s", err))
		http.Error(res, errorResponse, http.StatusInternalServerError)
		return
	}

	ctx := req.Context()

	// Rather than have a request body or query parameters, we get the necessary
	// information we need via the client sending a JSON message.
	var srequest signedRequest
	err = conn.ReadJSON(&srequest)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("error reading signed request: %s", err))
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	var payload model.LogsPayload
	err = json.Unmarshal(*srequest.Payload, &payload)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("unable to parse incoming request: %s", err))
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	// TODO: Check the actual signature

	ctx = system.AddJobIDToBaggage(ctx, payload.ClientID)

	// Get the job, check it exists and check it belongs to the same client
	job, err := s.jobStore.GetJob(ctx, payload.JobID)
	if err != nil {
		log.Ctx(ctx).Debug().Msgf("Missing job: %s", err)
		http.Error(res, bacerrors.ErrorToErrorResponse(err), http.StatusBadRequest)
		return
	}

	// We can compare the payload's client ID against the existing job's metadata
	// as we have confirmed the public key that the request was signed with matches
	// the client ID the request claims.
	if job.Metadata.ClientID != payload.ClientID {
		log.Ctx(ctx).Debug().Msgf("Mismatched ClientIDs for logs, existing job: %s and log request: %s",
			job.Metadata.ClientID, payload.ClientID)

		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("mismatched client id: %s", payload.ClientID))
		http.Error(res, errorResponse, http.StatusForbidden)
		return
	}

	// Ask the Compute node for a multiaddr where we can connect to a log server
	logRequest := requester.ReadLogsRequest{JobID: job.ID(), ExecutionID: payload.ExecutionID}
	response, err := s.requester.ReadLogs(ctx, logRequest)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("read logs failure: %s", err))
		http.Error(res, errorResponse, http.StatusBadRequest)
		return
	}

	opts := []libp2p.Option{
		libp2p.DisableRelay(),
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("failed to write to websocket connection: %s", err))
		http.Error(res, errorResponse, http.StatusInternalServerError)
		return
	}
	maddr, err := ma.NewMultiaddr(response.Address)
	if err != nil {
		return
	}
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return
	}

	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	addresses := host.Peerstore().Addrs(info.ID)
	if len(addresses) == 0 {
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	}

	stream, err := host.NewStream(ctx, info.ID, "/bacalhau/compute/logs/1.0.0")
	if err != nil {
		return
	}
	defer stream.Close()

	lsReq := LogStreamRequest{
		JobID:       payload.JobID,
		ExecutionID: payload.ExecutionID,
		WithHistory: payload.WithHistory,
	}

	err = json.NewEncoder(stream).Encode(lsReq)
	if err != nil {
		_ = stream.Reset()
		return
	}

	for {
		frame, err := logstream.NewDataFrameFromReader(stream)
		if err == io.EOF {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}
		if err != nil {
			log.Ctx(ctx).Error().Msgf("Stream read failure: %s", err)
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}

		msg := Msg{
			Tag:  uint8(frame.Tag),
			Data: string(frame.Data),
		}
		err = conn.WriteJSON(msg)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("websocket write failure: %s", err)
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	conn.Close()

	_ = stream.Reset()
	_ = stream.Close()
}
