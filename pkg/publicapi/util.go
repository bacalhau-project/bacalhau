package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func HTTPError(c echo.Context, err error, statusCode int) {
	log.Ctx(c.Request().Context()).Error().Err(err).Send()
	http.Error(c.Response(), bacerrors.ErrorToErrorResponse(err), statusCode)
}
