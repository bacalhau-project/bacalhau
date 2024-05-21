package local

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// PublisherSpec mainly exists as a form of documentation to indicate the local publisher spec does not contain params
type PublisherSpec struct{}

func NewSpecConfig() *models.SpecConfig {
	return &models.SpecConfig{
		Type:   models.PublisherLocal,
		Params: make(map[string]interface{}),
	}
}
