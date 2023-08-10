package repo

import (
	"fmt"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	memjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
)

// InitJobStore must be called after Init
func (fsr *FsRepo) InitJobStore(prefix string) (jobstore.Store, error) {
	if exists, err := fsr.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("repo is uninitialized, cannot create JobStore")
	}
	// load the compute nodes execution store config
	var storeCfg config.StorageConfig
	if err := config.GetConfigForKey(config.NodeRequesterJobStore, &storeCfg); err != nil {
		return nil, err
	}
	switch storeCfg.Type {
	case config.BoltDB:
		path := storeCfg.Path
		if path == "" {
			path = filepath.Join(fsr.path, fmt.Sprintf("%s-requester.db", prefix))
		}
		return boltjobstore.NewBoltJobStore(path)
	case config.InMemory:
		return memjobstore.NewInMemoryJobStore(), nil
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
}
