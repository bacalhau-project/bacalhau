package types

import (
	"fmt"
	"path/filepath"
)

const UserKeyFileName = "user_id.pem"

func (b Bacalhau) UserKeyPath() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, UserKeyFileName)
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

func (b Bacalhau) AuthTokensPath() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	return filepath.Join(b.DataDir, AuthTokensFileName), nil
}

const OrchestratorDirName = "orchestrator"

func (b Bacalhau) OrchestratorDir() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, OrchestratorDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting orchestrator path: %w", err)
	}
	return path, nil
}

const JobStoreFileName = "state_boltdb.db"

func (b Bacalhau) JobStoreFilePath() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	// make sure the parent dir exists first
	if _, err := b.OrchestratorDir(); err != nil {
		return "", fmt.Errorf("getting job store path: %w", err)
	}
	return filepath.Join(b.DataDir, OrchestratorDirName, JobStoreFileName), nil
}

const NetworkTransportDirName = "nats-store"

func (b Bacalhau) NetworkTransportDir() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, OrchestratorDirName, NetworkTransportDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting network transport path: %w", err)
	}
	return path, nil
}

const ComputeDirName = "compute"

func (b Bacalhau) ComputeDir() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, ComputeDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting compute path: %w", err)
	}
	return path, nil
}

const ExecutionDirName = "executions"

func (b Bacalhau) ExecutionDir() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, ComputeDirName, ExecutionDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting executions path: %w", err)
	}
	return path, nil
}

const ResultsStorageDir = "results"

func (b Bacalhau) ResultsStorageDir() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, ComputeDirName, ResultsStorageDir)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting results storage path: %w", err)
	}
	return path, nil
}

const PluginsDirName = "plugins"

func (b Bacalhau) PluginsDir() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	path := filepath.Join(b.DataDir, PluginsDirName)
	if err := ensureDir(path); err != nil {
		return "", fmt.Errorf("getting plugins path: %w", err)
	}
	return path, nil
}

const ExecutionStoreFileName = "state_boltdb.db"

func (b Bacalhau) ExecutionStoreFilePath() (string, error) {
	if b.DataDir == "" {
		return "", fmt.Errorf("data dir not set")
	}
	if _, err := b.ComputeDir(); err != nil {
		return "", fmt.Errorf("getting execution store path: %w", err)
	}
	return filepath.Join(b.DataDir, ComputeDirName, ExecutionStoreFileName), nil
}
