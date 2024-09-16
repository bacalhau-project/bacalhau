package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {
	var (
		code      int
		message   string
		errorCode string
		component string
	)

	switch e := err.(type) {

	case *models.BaseError:
		// If it is already our custom APIError, use its code and message
		code = e.HTTPStatusCode()
		message = e.Error()
		errorCode = string(e.Code())
		component = e.Component()

	case *echo.HTTPError:
		// This is needed, in case any other middleware throws an error. In
		// such a scenario we just use it as the error code and the message.
		// One such example being when request body size is larger then the max
		// size accepted
		code = e.Code
		message, _ = e.Message.(string)
		errorCode = string(models.InternalError)
		component = "APIServer"
		if c.Echo().Debug && e.Internal != nil {
			message += ". " + e.Internal.Error()
		}

	default:
		// In an ideal world this should never happen. We should always have are errors
		// from server as APIError. If output is this generic string, one should evaluate
		// and map it to APIError and send in appropriate message.= http.StatusInternalServerError
		code = http.StatusInternalServerError
		message = "Internal server error"
		errorCode = string(models.InternalError)
		component = "Unknown"

		if c.Echo().Debug {
			message += ". " + err.Error()
		}
	}

	// Don't override the status code if it is already been set.
	// This is something that is advised by ECHO framework.
	if !c.Response().Committed {
		apiError := apimodels.APIError{
			HTTPStatusCode: code,
			Message:        message,
			RequestID:      c.Request().Header.Get(echo.HeaderXRequestID),
			Code:           errorCode,
			Component:      component,
		}
		var responseErr error
		if c.Request().Method == http.MethodHead {
			responseErr = c.NoContent(code)
		} else {
			responseErr = c.JSON(code, apiError)
		}
		if responseErr != nil {
			log.Error().Err(responseErr).
				Str("original_error", err.Error()).
				Msg("Failed to send error response")
		}
	}

}
