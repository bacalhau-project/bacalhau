package types

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	Second = Duration(time.Second)
	Minute = Duration(time.Minute)
	Day    = Duration(time.Hour * 24)
)

// Default is the default configuration for a bacalhau node.
var Default = Bacalhau{
	DataDir: DefaultDataDir(),
	API: API{
		Host: "0.0.0.0",
		Port: 1234,
		Auth: AuthConfig{
			Methods: map[string]AuthenticatorConfig{
				"ClientKey": {
					Type: "challenge",
				},
			},
		},
	},
	NameProvider: "puuid",
	Orchestrator: Orchestrator{
		Enabled: true,
		Host:    "0.0.0.0",
		Port:    4222,
		NodeManager: NodeManager{
			DisconnectTimeout: Minute,
		},
		Scheduler: Scheduler{
			WorkerCount:          runtime.NumCPU(),
			HousekeepingInterval: 30 * Second,
			HousekeepingTimeout:  2 * Minute,
		},
		EvaluationBroker: EvaluationBroker{
			VisibilityTimeout: Minute,
			MaxRetryCount:     10,
		},
	},
	Compute: Compute{
		Enabled:       false,
		Orchestrators: []string{"nats://127.0.0.1:4222"},
		Heartbeat: Heartbeat{
			InfoUpdateInterval:     Minute,
			ResourceUpdateInterval: 30 * Second,
			Interval:               15 * Second,
		},
		AllocatedCapacity: ResourceScaler{
			CPU:    "70%",
			Memory: "70%",
			Disk:   "70%",
			GPU:    "100%",
		},
	},
	JobDefaults: JobDefaults{
		Batch: BatchJobDefaultsConfig{
			Priority: 0,
			Task: BatchTaskDefaultConfig{
				Resources: ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
		Ops: BatchJobDefaultsConfig{
			Priority: 0,
			Task: BatchTaskDefaultConfig{
				Resources: ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
		Daemon: LongRunningJobDefaultsConfig{
			Priority: 0,
			Task: LongRunningTaskDefaultConfig{
				Resources: ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
		Service: LongRunningJobDefaultsConfig{
			Priority: 0,
			Task: LongRunningTaskDefaultConfig{
				Resources: ResourcesConfig{
					CPU:    "500m",
					Memory: "1Gb",
				},
			},
		},
	},
	InputSources: InputSourcesConfig{
		ReadTimeout:   Duration(5 * time.Minute),
		MaxRetryCount: 3,
	},
	Engines: EngineConfig{
		Types: EngineConfigTypes{
			Docker: Docker{
				ManifestCache: DockerManifestCache{
					Size:    1000,
					TTL:     Duration(1 * time.Hour),
					Refresh: Duration(1 * time.Hour),
				}},
		},
	},
	Logging: Logging{
		Level:                "info",
		Mode:                 "default",
		LogDebugInfoInterval: 0,
	},
	UpdateConfig: UpdateConfig{
		Interval: Day,
	},
	WebUI: WebUI{
		Enabled: false,
		Listen:  "0.0.0.0:8438",
	},
}

const defaultBacalhauDir = ".bacalhau"

// DefaultDataDir determines the appropriate default directory for storing repository data.
// Priority order:
// 1. User's home directory with .bacalhau appended.
func DefaultDataDir() string {
	if userHome, err := os.UserHomeDir(); err == nil && userHome != "" {
		if expandedUserHome, err := filepath.Abs(userHome); err == nil {
			return filepath.Join(expandedUserHome, defaultBacalhauDir)
		}
	}
	return ""
}
