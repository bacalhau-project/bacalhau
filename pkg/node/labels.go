package node

import (
	"context"
	"runtime"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type RuntimeLabelsProvider struct{}

// GetLabels implements models.LabelsProvider.
func (*RuntimeLabelsProvider) GetLabels(context.Context) map[string]string {
	return map[string]string{
		"Operating-System": runtime.GOOS,
		"Architecture":     runtime.GOARCH,
	}
}

var _ models.LabelsProvider = (*RuntimeLabelsProvider)(nil)

type ConfigLabelsProvider struct {
	staticLabels map[string]string
}

func (p *ConfigLabelsProvider) GetLabels(context.Context) map[string]string {
	return p.staticLabels
}
