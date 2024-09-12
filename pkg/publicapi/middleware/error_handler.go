package middleware

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {

	var code int
	var message string
	var errorCode string
	var component string

	switch e := err.(type) {

	case *models.BaseError:
		// If it is already our custom APIError, use its code and message
		code = models.InferHTTPStatusCode(e.Code())
		message = e.Error()
		errorCode = string(e.Code())

	case *echo.HTTPError:
		// This is needed, in case any other middleware throws an error. In
		// such a scenario we just use it as the error code and the message.
		// One such example being when request body size is larger then the max
		// size accepted
		code = e.Code
		message = e.Message.(string)
		errorCode = string(models.InternalError)
		component = "Echo"

	default:
		// In an ideal world this should never happen. We should always have are errors
		// from server as APIError. If output is this generic string, one should evaluate
		// and map it to APIError and send in appropriate message.= http.StatusInternalServerError
		message = "internal server error"
		code = c.Response().Status
		errorCode = string(models.InternalError)
		component = "Unknown"

		if c.Echo().Debug {
			message = err.Error()
		}
	}

	requestID := c.Request().Header.Get(echo.HeaderXRequestID)

	// Don't override the status code if it is already been set.
	// This is something that is advised by ECHO framework.
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, apimodels.APIError{
				HTTPStatusCode: code,
				Message:        message,
				RequestID:      requestID,
				Code:           errorCode,
				Component:      component,
			})
		}
		if err != nil {
			log.Info().Msg("unable to send json response while handling error.")
		}
	}

}
