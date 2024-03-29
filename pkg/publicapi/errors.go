package publicapi

import (
	"net/http"

	"github.com/bacalhau-project/bacalhau/pkg/lib/bad"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

func toHTTPError(rawErr error) *echo.HTTPError {
	switch err := rawErr.(type) {
	case *echo.HTTPError:
		return err
	default:
		badErr := bad.ToError(err)
		return echo.NewHTTPError(getStatusCode(badErr), *badErr)
	}
}

func getStatusCode(err *bad.Error) int {
	switch err.Subject {
	case bad.ErrSubjectInput:
		return http.StatusBadRequest
	case bad.ErrSubjectDependency:
		return http.StatusServiceUnavailable
	case bad.ErrSubjectNone:
		// This error does not define a status code. Recurse downwards and find
		// the most appropriate one.
		return lo.Reduce(err.Errs, func(agg int, err bad.Error, _ int) int {
			return math.Min(getStatusCode(&err), agg)
		}, http.StatusInternalServerError)
	case bad.ErrSubjectInternal:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}

func handleAPIError(err error, c echo.Context) {
	c.Echo().DefaultHTTPErrorHandler(toHTTPError(err), c)
}
