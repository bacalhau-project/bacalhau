package orchestrator

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/labels"
)

// parseLabels parses labels params into a label selector.
func parseLabels(c echo.Context) (labels.Selector, error) {
	selector := labels.NewSelector()
	if c.QueryParams().Has("labels") {
		req, err := labels.ParseToRequirements(strings.Join(c.QueryParams()["labels"], ","))
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		selector = selector.Add(req...)
	}
	return selector, nil
}
