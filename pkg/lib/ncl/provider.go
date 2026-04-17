package ncl

type PublisherProvider interface {
	// GetPublisher returns a publisher for the given subject
	GetPublisher() (Publisher, error)
}

type LazyPublisherProvider struct {
	Provider PublisherProvider
}

// NewLazyPublisherProvider creates a new LazyPublisherProvider
func NewLazyPublisherProvider() *LazyPublisherProvider {
	return &LazyPublisherProvider{
		Provider: nil,
	}
}

// SetProvider sets the requester provider for the delegated requester provider
func (d *LazyPublisherProvider) SetProvider(provider PublisherProvider) {
	d.Provider = provider
}

// GetPublisher returns a publisher for the given subject
func (d *LazyPublisherProvider) GetPublisher() (Publisher, error) {
	if d.Provider == nil {
		return nil, nil
	}
	return d.Provider.GetPublisher()
}
