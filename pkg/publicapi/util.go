package publicapi

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
)

const defaultIndent = "  "

// UnescapedJSON writes a JSON response with unescaped HTML characters.
// This is useful for returning JSON responses that contain HTML, such as URLs with ampersands.
func UnescapedJSON(c echo.Context, code int, i interface{}) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
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
