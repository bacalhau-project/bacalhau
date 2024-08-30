package types

import (
	"fmt"
	"path/filepath"
)

const UserKeyFileName = "user_id.pem"

func (c Bacalhau) UserKeyPath() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(c.DataDir, UserKeyFileName)
	if exists, err := fileExists(path); err != nil {
		return "", fmt.Errorf("checking if user key exists: %w", err)
	} else if exists {
		return path, nil
	}
	if err := initUserIDKey(path); err != nil {
		return "", fmt.Errorf("creating user private key: %w", err)
	}
	return path, nil
}

const AuthTokensFileName = "tokens.json"

func (c Bacalhau) AuthTokensPath() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	return filepath.Join(c.DataDir, AuthTokensFileName), nil
}

const OrchestratorDirName = "orchestrator"

func (c Bacalhau) OrchestratorDir() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(c.DataDir, OrchestratorDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting orchestrator path: %w", err)
	}
	return path, nil
}

const JobStoreFileName = "state_boltdb.db"

func (c Bacalhau) JobStoreFilePath() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	// make sure the parent dir exists first
	if _, err := c.OrchestratorDir(); err != nil {
		return "", fmt.Errorf("getting job store path: %w", err)
	}
	return filepath.Join(c.DataDir, OrchestratorDirName, JobStoreFileName), nil
}

const NetworkTransportDirName = "nats-store"

func (c Bacalhau) NetworkTransportDir() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(c.DataDir, OrchestratorDirName, NetworkTransportDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting network transport path: %w", err)
	}
	return path, nil
}

const ComputeDirName = "compute"

func (c Bacalhau) ComputeDir() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(c.DataDir, ComputeDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting compute path: %w", err)
	}
	return path, nil
}

const ExecutionDirName = "executions"

func (c Bacalhau) ExecutionDir() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(c.DataDir, ComputeDirName, ExecutionDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting executions path: %w", err)
	}
	return path, nil
}

const PluginsDirName = "plugins"

func (c Bacalhau) PluginsDir() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(c.DataDir, PluginsDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting plugins path: %w", err)
	}
	return path, nil
}

const ExecutionStoreFileName = "state_boltdb.db"

func (c Bacalhau) ExecutionStoreFilePath() (string, error) {
	if c.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	if _, err := c.ComputeDir(); err != nil {
		return "", fmt.Errorf("getting execution store path: %w", err)
	}
	return filepath.Join(c.DataDir, ComputeDirName, ExecutionStoreFileName), nil
}
