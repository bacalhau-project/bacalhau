package node

import (
	"context"
	"runtime"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	LabelNodeName        = "name"
	LabelOperatingSystem = "os_type"
	LabelArchitecture    = "os_arch"
)

type RuntimeLabelsProvider struct{}

// NewRuntimeLabelsProvider creates a new LabelsProvider that returns the
// current operating system and architecture as labels.
func NewRuntimeLabelsProvider() *RuntimeLabelsProvider {
	return &RuntimeLabelsProvider{}
}

// GetLabels implements models.LabelsProvider.
func (*RuntimeLabelsProvider) GetLabels(context.Context) map[string]string {
	return map[string]string{
		LabelOperatingSystem: runtime.GOOS,
		LabelArchitecture:    runtime.GOARCH,
	}
}

var _ models.LabelsProvider = (*RuntimeLabelsProvider)(nil)

type ConfigLabelsProvider struct {
	staticLabels map[string]string
}

// NewConfigLabelsProvider creates a new LabelsProvider that returns the
// static labels provided in the configuration.
func NewConfigLabelsProvider(staticLabels map[string]string) *ConfigLabelsProvider {
	return &ConfigLabelsProvider{
		staticLabels: staticLabels,
	}
}

func (p *ConfigLabelsProvider) GetLabels(context.Context) map[string]string {
	return p.staticLabels
}

type NameLabelsProvider struct {
	name string
}

// NewNameLabelsProvider creates a new LabelsProvider that returns the
// node name as a label.
func NewNameLabelsProvider(name string) *NameLabelsProvider {
	return &NameLabelsProvider{
		name: name,
	}
}

func (p *NameLabelsProvider) GetLabels(context.Context) map[string]string {
	return map[string]string{
		LabelNodeName: p.name,
	}
}
