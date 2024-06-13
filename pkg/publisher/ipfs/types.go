package ipfs

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// PublisherSpec mainly exists as a form of documentation to indicate the IPFS publisher spec does not contain params
type PublisherSpec struct{}

func NewSpecConfig() *models.SpecConfig {
	return &models.SpecConfig{
		Type:   models.PublisherIPFS,
		Params: make(map[string]interface{}),
	}
}
