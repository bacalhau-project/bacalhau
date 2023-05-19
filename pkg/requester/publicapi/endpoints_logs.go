package publicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type logRequest = publicapi.SignedRequest[model.LogsPayload] //nolint:unused // Swagger wants this

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
	defer conn.Close()

	ctx := req.Context()

	// Rather than have a request body or query parameters, we get the necessary
	// information we need via the client sending a JSON message.
	var srequest json.RawMessage
	err = conn.ReadJSON(&srequest)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("error reading signed request: %s", err))
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, errorResponse))
		return
	}

	// This should not marshal badly given we just converted it from bytes
	// to a signedRequest. We have to convert it back to bytes as our
	// websocket connection isn't an io.Reader and so we can't ask it to
	// process the signature.
	buffer := bytes.NewReader(srequest)
	payload, err := publicapi.UnmarshalSigned[model.LogsPayload](ctx, buffer)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "failed to decode request"))
		return
	}

	ctx = system.AddJobIDToBaggage(ctx, payload.ClientID)

	// Get the job, check it exists and check it belongs to the same client
	job, err := s.jobStore.GetJob(ctx, payload.JobID)
	if err != nil {
		log.Ctx(ctx).Debug().Msgf("Missing job: %s", err)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error()))
		return
	}

	// We can compare the payload's client ID against the existing job's metadata
	// as we have confirmed the public key that the request was signed with matches
	// the client ID the request claims.
	if job.Metadata.ClientID != payload.ClientID {
		log.Ctx(ctx).Debug().Msgf("Mismatched ClientIDs for logs, existing job: %s and log request: %s",
			job.Metadata.ClientID, payload.ClientID)

		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("mismatched client id: %s", payload.ClientID))
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, errorResponse))
		return
	}

	// Ask the Compute node for a multiaddr where we can connect to a log server
	logRequest := requester.ReadLogsRequest{
		JobID:       job.ID(),
		ExecutionID: payload.ExecutionID,
		WithHistory: payload.WithHistory,
		Follow:      payload.Follow}
	response, err := s.requester.ReadLogs(ctx, logRequest)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("read logs failure: %s", err))
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, errorResponse))
		return
	}

	if response.ExecutionComplete {
		s.writeTerminatedJobOutput(ctx, conn, job.ID(), payload.ExecutionID)
		return
	}

	client, err := logstream.NewLogStreamClient(ctx, response.Address)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("logstream client create failure: %s", err))
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, errorResponse))
		return
	}
	defer client.Close()

	err = client.Connect(ctx, payload.ExecutionID, payload.WithHistory, payload.Follow)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("logstream connect failure: %s", err))
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, errorResponse))
		return
	}

	for {
		frame, err := client.ReadDataFrame(ctx)
		if err == io.EOF {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}
		if err != nil {
			log.Ctx(ctx).Error().Msgf("Stream read failure: %s", err)
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}

		err = s.writeDataFrame(ctx, conn, frame)
		if err != nil {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func (s *RequesterAPIServer) writeDataFrame(ctx context.Context, conn *websocket.Conn, frame logger.DataFrame) error {
	msg := Msg{
		Tag:  uint8(frame.Tag),
		Data: string(frame.Data),
	}

	err := conn.WriteJSON(msg)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("websocket write failure: %s", err)
	}
	return err
}

func (s *RequesterAPIServer) writeTerminatedJobOutput(
	ctx context.Context,
	conn *websocket.Conn,
	jobID string,
	executionID string) {
	jobState, err := s.jobStore.GetJobState(ctx, jobID)
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
		return
	}

	for _, exec := range jobState.Executions {
		if exec.ComputeReference == executionID {
			if exec.RunOutput.STDOUT != "" {
				df := logger.DataFrame{
					Tag:  logger.StdoutStreamTag,
					Size: len(exec.RunOutput.STDOUT),
					Data: []byte(exec.RunOutput.STDOUT),
				}
				_ = s.writeDataFrame(ctx, conn, df)
			}

			if exec.RunOutput.STDERR != "" {
				df := logger.DataFrame{
					Tag:  logger.StderrStreamTag,
					Size: len(exec.RunOutput.STDERR),
					Data: []byte(exec.RunOutput.STDERR),
				}
				_ = s.writeDataFrame(ctx, conn, df)
			}
		}
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
