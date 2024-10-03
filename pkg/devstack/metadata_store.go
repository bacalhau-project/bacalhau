package devstack

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/bacalhau-project/bacalhau/pkg/node"
)

// MetadataStore is a simple in-memory implementation of repo's system_metadata store
// that is useful for testing and development.
type MetadataStore struct {
	instanceID      string
	lastUpdateCheck time.Time
	mu              sync.Mutex
}

func NewMetadataStore() *MetadataStore {
	return &MetadataStore{}
}

func (m *MetadataStore) ReadLastUpdateCheck() (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastUpdateCheck, nil
}

func (m *MetadataStore) WriteLastUpdateCheck(time time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastUpdateCheck = time
	return nil
}

func (m *MetadataStore) InstanceID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.instanceID == "" {
		m.instanceID = uuid.NewString()
	}
	return m.instanceID
}

// compile time check whether the MetadataStore implements the node.MetadataStore interface
var _ node.MetadataStore = &MetadataStore{}
