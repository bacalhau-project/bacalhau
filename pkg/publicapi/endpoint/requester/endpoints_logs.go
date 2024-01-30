package requester

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/signatures"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Msg struct {
	Tag          uint8
	Data         string
	ErrorMessage string
}

// @ID				pkg/requester/publicapi/logs
// @Summary		Displays the logs for a current job/execution
// @Description	Shows the output from the job specified by `id` as long as that job belongs to `client_id`.
// @Description	The output will be continuous until either, the client disconnects or the execution completes.
// @Tags			Job
// @Accept			json
// @Produce		json
// @Param			LogRequest	body		legacymodels.LogRequest	true	" "
// @Success		200			{object}	string
// @Failure		400			{object}	string
// @Failure		401			{object}	string
// @Failure		403			{object}	string
// @Failure		500			{object}	string
// @Router			/api/v1/requester/logs [post]
//
//nolint:funlen,gocyclo
func (s *Endpoint) logs(c echo.Context) error {
	var upgrader = websocket.Upgrader{}
	conn, err := upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("failed to upgrade websocket connection: %s", err))
		http.Error(c.Response(), errorResponse, http.StatusInternalServerError)
		return nil
	}
	defer conn.Close()

	ctx := c.Request().Context()

	// Rather than have a request body or query parameters, we get the necessary
	// information we need via the client sending a JSON message.
	var srequest json.RawMessage
	err = conn.ReadJSON(&srequest)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("error reading signed request: %s", err))
		s.writeErrorMessage(ctx, conn, errorResponse)
		return nil
	}

	// This should not marshal badly given we just converted it from bytes
	// to a signedRequest. We have to convert it back to bytes as our
	// websocket connection isn't an io.Reader and so we can't ask it to
	// process the signature.
	buffer := bytes.NewReader(srequest)
	payload, err := signatures.UnmarshalSigned[model.LogsPayload](ctx, buffer)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.New("failed to decode request"))
		s.writeErrorMessage(ctx, conn, errorResponse)
		return nil
	}

	ctx = system.AddJobIDToBaggage(ctx, payload.ClientID)

	// Get the job, check it exists and check it belongs to the same client
	job, err := s.jobStore.GetJob(ctx, payload.JobID)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("failed to find job: %s", payload.JobID))
		s.writeErrorMessage(ctx, conn, errorResponse)
		return nil
	}

	// Ask the Compute node for a multiaddr where we can connect to a log server
	logRequest := requester.ReadLogsRequest{
		JobID:       job.ID,
		ExecutionID: payload.ExecutionID,
		WithHistory: payload.WithHistory,
		Follow:      payload.Follow}
	responseCh, err := s.requester.ReadLogs(ctx, logRequest)
	if err != nil {
		errorResponse := bacerrors.ErrorToErrorResponse(errors.Errorf("read logs failure: %s", err))
		s.writeErrorMsg(ctx, conn, errorResponse)
		return nil
	}

	for {
		response, ok := <-responseCh
		if !ok {
			break
		}
		err = s.writeMsg(ctx, conn, response)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msgf("websocket write failure")
			break
		}
		if response.EOF {
			break
		}
	}

	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return nil
}

func (s *Endpoint) writeErrorMessage(ctx context.Context, conn *websocket.Conn, errorMsg string) {
	msg := Msg{
		ErrorMessage: errorMsg,
	}

	err := conn.WriteJSON(msg)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("websocket write failure sending error: %s", err)
	}
}

func (s *Endpoint) writeErrorMsg(ctx context.Context, conn *websocket.Conn, errorMsg string) {
	_ = s.writeMsg(ctx, conn, models.ExecutionLog{
		Error: errorMsg,
	})
}

func (s *Endpoint) writeMsg(ctx context.Context, conn *websocket.Conn, msg any) error {
	err := conn.WriteJSON(msg)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("websocket write failure: %s", err)
	}
	return err
}
