package middleware

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {

	log.Info().Msg("HELLO THIS IS CALLED")
	var code int
	var message string

	switch e := err.(type) {

	case *apimodels.APIError:
		// If it is already our custom APIError, use its code and message
		code = e.HTTPStatusCode
		message = e.Message

	default:
		// In an ideal world this should never happen. We should always have are errors
		// from server as APIError. If output is this generic string, one should evaluate
		// and map it to APIError and send in appropriate message.= http.StatusInternalServerError
		message = "internal server error"
	}

	// Don't override the status code if it is already been set.
	// This is something that is advised by ECHO framework.
	if !c.Response().Committed {

		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, apimodels.APIError{
				HTTPStatusCode: code,
				Message:        message,
			})
		}
		if err != nil {
			log.Info().Msg("unable to send json response while handling error.")
		}
	}

}
