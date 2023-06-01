package distributed

type Option func(*DistributedObjectStore)

func WithPeers(peers []string) Option {
	return func(d *DistributedObjectStore) {
	}
}
