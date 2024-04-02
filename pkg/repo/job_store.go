package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	memjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/rs/zerolog/log"
)

// InitJobStore must be called after Init and uses the configuration to create a
// new JobStore for the requester node.  Where BoltDB is chosen, and no path is specified,
// then the database will be created in the repo in a folder labeledafter the node ID.
// For example:
// `~/.bacalhau/Qmd1BEyR4RsLdYTEym1YxxaeXFdwWCMANYN7XCcpPYbTRs-requester/jobs.db`
func (fsr *FsRepo) InitJobStore(ctx context.Context, prefix string) (jobstore.Store, error) {
	if exists, err := fsr.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("repo is uninitialized, cannot create JobStore")
	}
	// load the compute nodes execution store config
	var storeCfg types.JobStoreConfig
	if err := config.ForKey(types.NodeRequesterJobStore, &storeCfg); err != nil {
		return nil, err
	}
	switch storeCfg.Type {
	case types.BoltDB:
		path := storeCfg.Path
		if path == "" {
			directory := filepath.Join(fsr.path, fmt.Sprintf("%s-requester", prefix))
			if err := os.MkdirAll(directory, os.ModePerm); err != nil {
				return nil, err
			}

			path = filepath.Join(directory, "jobs.db")
		}

		log.Ctx(ctx).Debug().Str("Path", path).Msg("creating boltdb backed jobstore")
		return boltjobstore.NewBoltJobStore(path)
	case types.InMemory:
		log.Ctx(ctx).Debug().Msg("creating inmemory backed jobstore")
		return memjobstore.NewInMemoryJobStore(), nil
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
}
