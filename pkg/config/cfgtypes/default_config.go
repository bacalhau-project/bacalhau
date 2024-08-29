package cfgtypes

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	Second = Duration(time.Second)
	Minute = Duration(time.Minute)
	Day    = Duration(time.Hour * 24)
)

// Default is the default configuration for a bacalhau node.
var Default = Bacalhau{
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
	DataDir:      getDefaultDataDir(),
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
				Publisher: DefaultPublisherConfig{
					Type: models.PublisherLocal,
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
				Publisher: DefaultPublisherConfig{
					Type: models.PublisherLocal,
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
				Publisher: DefaultPublisherConfig{
					Type: models.PublisherLocal,
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
				Publisher: DefaultPublisherConfig{
					Type: models.PublisherLocal,
				},
			},
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
	DefaultPublisher: DefaultPublisherConfig{
		Type: models.PublisherLocal,
	},
}

const defaultBacalhauDir = ".bacalhau"

// getDefaultDataDir determines the appropriate default directory for storing repository data.
// Priority order:
// 1. If the environment variable BACALHAU_DIR is set and non-empty, use it.
// 3. User's home directory with .bacalhau appended.
// 4. User-specific configuration directory with .bacalhau appended.
// 5. If all above fail, use .bacalhau in the current directory.
// The function logs any errors encountered during the process and always returns a usable path.
func getDefaultDataDir() string {
	if repoDir, set := os.LookupEnv("BACALHAU_DIR"); set && repoDir != "" {
		return repoDir
	}

	if userHome, err := os.UserHomeDir(); err == nil {
		return filepath.Join(userHome, defaultBacalhauDir)
	}

	if userDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(userDir, defaultBacalhauDir)
	}

	return defaultBacalhauDir
}
