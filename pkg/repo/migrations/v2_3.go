package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

// V2Migration updates the repo so that nodeID is no longer part of the execution and job store paths.
// It does the following:
// - Generates and persists the nodeID in the config if it is missing, which is the case for v2 repos
// - Adds the execution and job store paths to the config if they are missing, which is the case for v3 repos
// - Renames the execution and job store directories to the new name if they exist
var V2Migration = repo.NewMigration(
	repo.RepoVersion2,
	repo.RepoVersion3,
	func(r repo.FsRepo) error {
		v, fileCfg, err := readConfig(r)
		if err != nil {
			return err
		}
		repoPath, err := r.Path()
		if err != nil {
			return err
		}
		// we load the config to resolve the libp2p node id. Loading the config this way will also
		// use default values, args and env vars to fill in the config, so we can be sure we are
		// reading the correct libp2p key in case the user is overriding the default value.
		resolvedCfg, err := config.Load(repoPath)
		if err != nil {
			return err
		}
		libp2pNodeID, err := getLibp2pNodeID()
		if err != nil {
			return err
		}

		doWrite := false
		var logMessage strings.Builder
		set := func(key string, value interface{}) {
			v.Set(key, value)
			logMessage.WriteString(fmt.Sprintf("\n%s:\t%v", key, value))
			doWrite = true
		}

		emptyConfig := types.JobStoreConfig{}
		if fileCfg.Node.Compute.ExecutionStore == emptyConfig {
			// persist the execution store in the repo
			executionStore := resolvedCfg.Node.Compute.ExecutionStore

			// if execution store already exist with nodeID, then rename it to the new name
			legacyStoreName := filepath.Join(repoPath, libp2pNodeID+"-compute")
			newStorePath := filepath.Dir(executionStore.Path)
			if _, err := os.Stat(legacyStoreName); err == nil {
				if err := os.Rename(legacyStoreName, newStorePath); err != nil {
					return err
				}
			} else if err = os.MkdirAll(newStorePath, util.OS_USER_RWX); err != nil {
				return err
			}
			set(types.NodeComputeExecutionStore, executionStore)
		}

		if fileCfg.Node.Requester.JobStore == emptyConfig {
			// persist the job store in the repo
			jobStore := resolvedCfg.Node.Requester.JobStore

			// if job store already exist with nodeID, then rename it to the new name
			legacyStoreName := filepath.Join(repoPath, libp2pNodeID+"-requester")
			newStorePath := filepath.Dir(jobStore.Path)
			if _, err := os.Stat(legacyStoreName); err == nil {
				if err := os.Rename(legacyStoreName, newStorePath); err != nil {
					return err
				}
			} else if err = os.MkdirAll(newStorePath, util.OS_USER_RWX); err != nil {
				return err
			}
			set(types.NodeRequesterJobStore, jobStore)
		}

		if fileCfg.Node.Name == "" {
			set(types.NodeName, libp2pNodeID)
		}

		if doWrite {
			return v.WriteConfig()
		}
		return nil
	})
