package distributed

import (
	"os"
)

type Option func(*DistributedObjectStore)

func WithPeers(peers []string) Option {
	return func(d *DistributedObjectStore) {
	}
}

func WithTestConfig() Option {
	return func(d *DistributedObjectStore) {
		d.dataDir, _ = os.MkdirTemp("", "")
		d.cm.RegisterCallback(func() error {
			os.RemoveAll(d.dataDir)
			return nil
		})
	}
}
