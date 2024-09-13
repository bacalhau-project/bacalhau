package orchestrator

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/version"
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

// backwardCompatibleHistoryIfNecessary sets the state change fields to non-nil for backward compatibility
// with v1.4.x clients. Otherwise, nil exceptions will be thrown when the client tries to describe job or list history.
//
//nolint:staticcheck
func backwardCompatibleHistoryIfNecessary(c echo.Context, items []*models.JobHistory) {
	clientVersionStr := c.Request().Header.Get(apimodels.HTTPHeaderBacalhauGitVersion)

	// If the client version is not explicit or is greater than or equal to v1.5, we don't need to do anything.
	if !version.IsVersionExplicit(clientVersionStr) || clientVersionStr >= "v1.5." {
		return
	}

	// If the client version is not explicit, or is less than v1.5, we need to set the state change fields to nil.
	for i := range items {
		history := items[i]
		if history.Type == models.JobHistoryTypeJobLevel {
			history.JobState = &models.StateChange[models.JobStateType]{}
		} else {
			history.ExecutionState = &models.StateChange[models.ExecutionStateType]{}
		}
	}
}
