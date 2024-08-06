package migrations

import (
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// Old IPFS config that are not valid anymore
const legacyIPFSSwarmAddressesKey = "Node.IPFS.SwarmAddresses"
const legacyBootstrapAddressesKey = "Node.BootstrapAddresses"

var V1Migration = repo.NewMigration(
	repo.Version1,
	repo.Version2,
	func(r repo.FsRepo) error {
		configExist, err := configExists(r)
		if err != nil {
			return err
		}
		if !configExist {
			return nil
		}
		v, _, err := readConfig(r)
		if err != nil {
			return err
		}

		// this migration removes any IPFS swarm peers or Bootstrap peers that are incorrect from the v1.0.4 upgrade.
		// if no incorrect values are present they are left as is.
		doWrite := false
		if v.Get(legacyIPFSSwarmAddressesKey) != nil {
			v.Set(legacyIPFSSwarmAddressesKey, []string{})
			doWrite = true
		}
		if v.Get(legacyBootstrapAddressesKey) != nil {
			v.Set(legacyBootstrapAddressesKey, []string{})
			doWrite = true
		}
		if doWrite {
			return v.WriteConfig()
		}
		return nil
	})
