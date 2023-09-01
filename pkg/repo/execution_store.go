package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inlocalstore"
	memcomputestore "github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// InitExecutionStore must be called after Init and uses the configuration to create a
// new ExecutionStore.  Where BoltDB is chosen, and no path is specified, then the database
// will be created in the repo in a folder labeledafter the node ID.  For example:
// `~/.bacalhau/Qmd1BEyR4RsLdYTEym1YxxaeXFdwWCMANYN7XCcpPYbTRs-compute/executions.db`
func (fsr *FsRepo) InitExecutionStore(ctx context.Context, prefix string) (store.ExecutionStore, error) {
	if exists, err := fsr.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("repo is uninitialized, cannot create ExecutionStore")
	}
	stateRootDir := filepath.Join(fsr.path, fmt.Sprintf("%s-compute", prefix))
	if err := os.MkdirAll(stateRootDir, os.ModePerm); err != nil {
		return nil, err
	}
	// load the compute nodes execution store config
	var storeCfg types.StorageConfig
	if err := config.ForKey(types.NodeComputeExecutionStore, &storeCfg); err != nil {
		return nil, err
	}
	var (
		store store.ExecutionStore
		err   error
	)
	switch storeCfg.Type {
	case types.BoltDB:
		path := storeCfg.Path
		if path == "" {
			path = filepath.Join(stateRootDir, "executions.db")
		}

		store, err = boltdb.NewStore(ctx, path)
		if err != nil {
			return nil, err
		}
	case types.InMemory:
		store = memcomputestore.NewStore()
	default:
		return nil, fmt.Errorf("unknown JobStore type: %s", storeCfg.Type)
	}
	return inlocalstore.NewPersistentExecutionStore(inlocalstore.PersistentJobStoreParams{
		Store:   store,
		RootDir: stateRootDir,
	})
}
