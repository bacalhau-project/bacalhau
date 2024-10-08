package migrations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

// V2Migration updates the repo so that nodeID is no longer part of the execution and job store paths.
// It does the following:
// - Generates and persists the nodeID in the config if it is missing, which is the case for v2 repos
// - Adds the execution and job store paths to the config if they are missing, which is the case for v3 repos
// - Renames the execution and job store directories to the new name if they exist
var V2Migration = repo.NewMigration(
	repo.Version2,
	repo.Version3,
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
		opts := []config_legacy.Option{config_legacy.WithValues(viper.AllSettings())}
		if _, err := os.Stat(filepath.Join(repoPath, config_legacy.FileName)); err != nil {
			// if the config doesn't exist that's okay, just means we read a repo without a config in it.
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("loading config from repo: %w", err)
			}
		} else {
			opts = append(opts, config_legacy.WithPaths(filepath.Join(repoPath, config_legacy.FileName)))
		}
		c, err := config_legacy.New(
			opts...,
		)
		if err != nil {
			return err
		}
		// modify the config with default paths required by the repo
		r.EnsureRepoPathsConfigured(c)
		resolvedCfg, err := c.Current()
		if err != nil {
			return err
		}
		libp2pNodeID, err := getLibp2pNodeID(repoPath)
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

		if fileCfg.Node.Compute.ExecutionStore.Path == "" {
			// persist the execution store in the repo
			executionStore := resolvedCfg.Node.Compute.ExecutionStore

			// handle an edge case where config.yaml has store config entry but no path,
			// which will override the resolved config path and make it empty as well
			if executionStore.Path == "" {
				executionStore.Path = filepath.Join(repoPath, "compute_store", "executions.db")
			}

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
			set(legacy_types.NodeComputeExecutionStore, executionStore)
		}

		if fileCfg.Node.Requester.JobStore.Path == "" {
			// persist the job store in the repo
			jobStore := resolvedCfg.Node.Requester.JobStore

			// handle an edge case where config.yaml has store config entry but no path,
			// which will override the resolved config path and make it empty as well
			if jobStore.Path == "" {
				jobStore.Path = filepath.Join(repoPath, "orchestrator_store", "jobs.db")
			}

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
			set(legacy_types.NodeRequesterJobStore, jobStore)
		}

		if fileCfg.Node.Name == "" {
			set(legacy_types.NodeName, libp2pNodeID)
		}

		if doWrite {
			return v.WriteConfig()
		}
		return nil
	})
