package types

import (
	"path/filepath"
)

//go:generate go run gen_paths/generate.go ./
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	Repo    string        `yaml:"Repo,omitempty"`
	Node    NodeConfig    `yaml:"Node"`
	User    UserConfig    `yaml:"User"`
	Metrics MetricsConfig `yaml:"Metrics"`
	Update  UpdateConfig  `yaml:"UpdateConfig"`
	Auth    AuthConfig    `yaml:"Auth"`
}

type UserConfig struct {
	// KeyPath is deprecated
	// Deprecated: use repo package
	KeyPath        string `yaml:"KeyPath"`
	InstallationID string `yaml:"InstallationID"`
}

type MetricsConfig struct {
	EventTracerPath string `yaml:"EventTracerPath"`
}

type UpdateConfig struct {
	SkipChecks     bool     `yaml:"SkipChecks"`
	CheckFrequency Duration `yaml:"CheckFrequency"`
}

const UserKeyFileName = "user_id.pem"

func (c BacalhauConfig) UserKeyPath() string {
	return filepath.Join(c.Repo, UserKeyFileName)
}

const AuthTokensFileName = "tokens.json"

func (c BacalhauConfig) AuthTokensPath() string {
	return filepath.Join(c.Repo, AuthTokensFileName)
}

const OrchestratorDirName = "orchestrator_store"

func (c BacalhauConfig) OrchestratorDir() string {
	return filepath.Join(c.Repo, OrchestratorDirName)
}

const JobStoreFileName = "jobs.db"

func (c BacalhauConfig) JobStorePath() string {
	return filepath.Join(c.Repo, JobStoreFileName)
}

const NetworkTransportDirName = OrchestratorDirName + "/" + "nats-store"

func (c BacalhauConfig) NetworkTransportDir() string {
	return filepath.Join(c.Repo, NetworkTransportDirName)
}

const ComputeDirName = "compute_store"

func (c BacalhauConfig) ComputeDir() string {
	return filepath.Join(c.Repo, ComputeDirName)
}

const ExecutionDirName = ComputeDirName + "/" + "executions"

func (c BacalhauConfig) ExecutionDir() string {
	return filepath.Join(c.Repo, ExecutionDirName)
}

const EnginePluginsDirName = ComputeDirName + "/" + "plugins" + "/" + "engines"

func (c BacalhauConfig) EnginePluginsDir() string {
	return filepath.Join(c.Repo, EnginePluginsDirName)
}

const ExecutionStoreFileName = "executions.db"

func (c BacalhauConfig) ExecutionStorePath() string {
	return filepath.Join(c.Repo, ExecutionStoreFileName)
}
