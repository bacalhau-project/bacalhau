package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		c.Echo().Logger.Warn("error handler skipped as response already committed")
		return
	}

	var apiError *apimodels.APIError

	switch e := err.(type) {
	case bacerrors.Error:
		apiError = apimodels.FromBacError(e)

	case *echo.HTTPError:
		// This is needed, in case any other middleware throws an error. In
		// such a scenario we just use it as the error code and the message.
		// One such example being when request body size is larger then the max
		// size accepted
		apiError = &apimodels.APIError{
			HTTPStatusCode: e.Code,
			Code:           string(bacerrors.InternalError),
			Message:        e.Message.(string),
			Component:      "APIServer",
		}
		if c.Echo().Debug && e.Internal != nil {
			apiError.Message += ". " + e.Internal.Error()
		}

	default:
		// In an ideal world this should never happen. We should always have are errors
		// from server as APIError. If output is this generic string, one should evaluate
		// and map it to APIError and send in appropriate message.= http.StatusInternalServerError
		apiError = &apimodels.APIError{
			HTTPStatusCode: http.StatusInternalServerError,
			Code:           string(bacerrors.InternalError),
			Message:        "Internal server error",
			Component:      "Unknown",
		}
		if c.Echo().Debug {
			apiError.Message += ". " + err.Error()
		}
	}

	apiError.RequestID = c.Request().Header.Get(echo.HeaderXRequestID)
	var responseErr error
	if c.Request().Method == http.MethodHead {
		responseErr = c.NoContent(apiError.HTTPStatusCode)
	} else {
		responseErr = c.JSON(apiError.HTTPStatusCode, apiError)
	}
	if responseErr != nil {
		log.Error().Err(responseErr).
			Str("original_error", err.Error()).
			Msg("Failed to send error response")
	}
}
