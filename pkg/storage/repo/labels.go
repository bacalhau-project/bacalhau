package repo

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type labelsProvider struct{}

func NewLabelsProvider() models.LabelsProvider {
	return labelsProvider{}
}

// GetLabels implements models.LabelsProvider.
func (labelsProvider) GetLabels(ctx context.Context) map[string]string {
	return map[string]string{
		"git-lfs": fmt.Sprint(checkGitLFS() != nil),
	}
}
