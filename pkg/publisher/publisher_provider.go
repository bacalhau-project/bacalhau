package publisher

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

// MappedPublisherProvider is a simple publisher repo that selects a publisher based on the job's publisher type.
type MappedPublisherProvider struct {
	publishers                    map[model.Publisher]Publisher
	publishersInstalledCache      map[model.Publisher]bool
	publishersInstalledCacheMutex sync.Mutex
}

func NewMappedPublisherProvider(publishers map[model.Publisher]Publisher) *MappedPublisherProvider {
	return &MappedPublisherProvider{
		publishers:               publishers,
		publishersInstalledCache: map[model.Publisher]bool{},
	}
}

func (p *MappedPublisherProvider) GetPublisher(ctx context.Context, publisherType model.Publisher) (Publisher, error) {
	publisher, ok := p.publishers[publisherType]
	if !ok {
		return nil, fmt.Errorf(
			"no matching publisher found on this server: %s", publisherType)
	}

	p.publishersInstalledCacheMutex.Lock()
	defer p.publishersInstalledCacheMutex.Unlock()

	// cache it being installed so we're not hammering it
	// TODO: we should evict the cache in case an installed publisher gets uninstalled, or vice versa
	installed, ok := p.publishersInstalledCache[publisherType]
	var err error
	if !ok {
		installed, err = publisher.IsInstalled(ctx)
		if err != nil {
			return nil, err
		}
		p.publishersInstalledCache[publisherType] = installed
	}

	if !installed {
		return nil, fmt.Errorf("publisher is not installed: %s", publisherType)
	}

	return publisher, nil
}

func (p *MappedPublisherProvider) HasPublisher(ctx context.Context, publisher model.Publisher) bool {
	_, err := p.GetPublisher(ctx, publisher)
	return err == nil
}
