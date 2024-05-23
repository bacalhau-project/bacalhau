package migrations

import (
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// V3Migration deletes the network store (NATS KV store) so that it may recreated once the node starts.
// It deletes the store since in v1.3.0 the store contains models.NodeInfo and in v1.3.1 the store
// contains models.NodeState and these types are incompatible with each other.
// Full details may be found in https://github.com/bacalhau-project/bacalhau/issues/4024
var V3Migration = repo.NewMigration(
	repo.RepoVersion3,
	repo.RepoVersion4,
	func(r repo.FsRepo) error {
		repoPath, err := r.Path()
		if err != nil {
			return err
		}
		resolvedCfg, err := config.Load(repoPath)
		if err != nil {
			return err
		}
		return os.RemoveAll(resolvedCfg.Node.Network.StoreDir)
	},
)
