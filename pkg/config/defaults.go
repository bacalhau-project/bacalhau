package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// Default is the default configuration for a bacalhau node.
var Default = types.Bacalhau{
	DataDir: defaultDataDir(),
	API: types.API{
		Host: "0.0.0.0",
		Port: 1234,
		Auth: types.AuthConfig{
			Methods: map[string]types.AuthenticatorConfig{
				"ClientKey": {
					Type: "challenge",
				},
			},
		},
	},
	NameProvider: "puuid",
	Orchestrator: types.Orchestrator{
		Enabled: false,
		Host:    "0.0.0.0",
		Port:    4222,
		NodeManager: types.NodeManager{
			DisconnectTimeout: types.Minute,
		},
		Scheduler: types.Scheduler{
			WorkerCount:          runtime.NumCPU(),
			QueueBackoff:         types.Minute,
			HousekeepingInterval: 30 * types.Second,
			HousekeepingTimeout:  2 * types.Minute,
		},
		EvaluationBroker: types.EvaluationBroker{
			VisibilityTimeout: types.Minute,
			MaxRetryCount:     10,
		},
	},
	Compute: types.Compute{
		Enabled:       false,
		Orchestrators: []string{"nats://127.0.0.1:4222"},
		Heartbeat: types.Heartbeat{
			InfoUpdateInterval: types.Minute,
			Interval:           15 * types.Second,
		},
		AllocatedCapacity: types.ResourceScaler{
			CPU:    "70%",
			Memory: "70%",
			Disk:   "70%",
			GPU:    "100%",
		},
	},
	JobDefaults: types.JobDefaults{
		Batch: types.BatchJobDefaultsConfig{
			Priority: 0,
			Task: types.BatchTaskDefaultConfig{
				Resources: types.ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
		Ops: types.BatchJobDefaultsConfig{
			Priority: 0,
			Task: types.BatchTaskDefaultConfig{
				Resources: types.ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
		Daemon: types.LongRunningJobDefaultsConfig{
			Priority: 0,
			Task: types.LongRunningTaskDefaultConfig{
				Resources: types.ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
		Service: types.LongRunningJobDefaultsConfig{
			Priority: 0,
			Task: types.LongRunningTaskDefaultConfig{
				Resources: types.ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
	},
	InputSources: types.InputSourcesConfig{
		ReadTimeout:   5 * types.Minute,
		MaxRetryCount: 3,
	},
	Engines: types.EngineConfig{
		Types: types.EngineConfigTypes{
			Docker: types.Docker{
				ManifestCache: types.DockerManifestCache{
					Size:    1000,
					TTL:     types.Duration(1 * time.Hour),
					Refresh: types.Duration(1 * time.Hour),
				}},
		},
	},
	Publishers: types.PublishersConfig{
		Types: types.PublisherTypes{
			Local: types.LocalPublisher{
				Address: "127.0.0.1",
				Port:    6001,
			},
		},
	},
	JobAdmissionControl: types.JobAdmissionControl{
		Locality: models.Anywhere,
	},
	Logging: types.Logging{
		Level:                "info",
		Mode:                 "default",
		LogDebugInfoInterval: 30 * types.Second,
	},
	UpdateConfig: types.UpdateConfig{
		Interval: types.Day,
	},
	WebUI: types.WebUI{
		Enabled: false,
		Listen:  "0.0.0.0:8438",
	},
}

var testOverrides = types.Bacalhau{
	API: types.API{
		Port: -1,
		Auth: types.AuthConfig{},
	},
	Orchestrator: types.Orchestrator{
		Port: -1,
		NodeManager: types.NodeManager{
			DisconnectTimeout: types.Duration(30 * time.Second),
		},
		Scheduler: types.Scheduler{
			WorkerCount:          3,
			HousekeepingTimeout:  types.Duration(5 * time.Second),
			HousekeepingInterval: 1 * types.Second,
		},
		EvaluationBroker: types.EvaluationBroker{
			VisibilityTimeout: types.Duration(5 * time.Second),
			MaxRetryCount:     3,
		},
	},
	Compute: types.Compute{
		Heartbeat: types.Heartbeat{
			Interval: 5 * types.Second,
		},
	},
	Publishers: types.PublishersConfig{
		Types: types.PublisherTypes{
			Local: types.LocalPublisher{
				Port: -1,
			},
		},
	},
	Logging: types.Logging{
		Level: "debug",
	},
	UpdateConfig: types.UpdateConfig{
		Interval: -1,
	},
	DisableAnalytics: true,
}

// NewTestConfig returns a new configuration with the default values for testing.
func NewTestConfig() (types.Bacalhau, error) {
	cfg, err := Default.MergeNew(testOverrides)
	if err != nil {
		return types.Bacalhau{}, err
	}

	// Create a new temporary directory under the system temp directory
	tempDir, err := os.MkdirTemp("", "bacalhau-test-*")
	if err != nil {
		return types.Bacalhau{}, err
	}

	cfg.DataDir = tempDir
	return cfg, nil
}

func NewTestConfigWithOverrides(overrides types.Bacalhau) (types.Bacalhau, error) {
	cfg, err := NewTestConfig()
	if err != nil {
		return types.Bacalhau{}, err
	}

	return cfg.MergeNew(overrides)
}

// defaultDataDir determines the appropriate default directory for storing repository data.
// Priority order:
// 1. User's home directory with .bacalhau appended.
func defaultDataDir() string {
	if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
		if expandedUserHome, err := filepath.Abs(userHome); err == nil {
			return filepath.Join(expandedUserHome, ".bacalhau")
		}
	}
	return ""
}
