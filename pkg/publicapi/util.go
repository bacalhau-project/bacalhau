package publicapi

import (
	"encoding/json"
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

const defaultIndent = "  "

func HTTPError(c echo.Context, err error, statusCode int) {
	log.Ctx(c.Request().Context()).Error().Err(err).Send()
	http.Error(c.Response(), bacerrors.ErrorToErrorResponse(err), statusCode)
}

// UnescapedJSON writes a JSON response with unescaped HTML characters.
// This is useful for returning JSON responses that contain HTML, such as URLs with ampersands.
func UnescapedJSON(c echo.Context, code int, i interface{}) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(code)

	indent := ""
	if _, pretty := c.QueryParams()["pretty"]; pretty {
		indent = defaultIndent
	}
	encoder := json.NewEncoder(c.Response())
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", indent)
	return encoder.Encode(i)
}
