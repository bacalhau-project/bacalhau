package configv2

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	Second = types.Duration(time.Second)
	Minute = types.Duration(time.Minute)
	Day    = types.Duration(time.Hour * 24)
)

// Default is the default configuration for a bacalhau node.
var Default = types.Bacalhau{
	API: types.API{
		Address: "http://0.0.0.0:1234",
		Auth: types.AuthConfig{
			Methods: map[string]types.AuthenticatorConfig{
				"ClientKey": {
					Type: string(authn.MethodTypeChallenge),
				},
			},
		},
	},
	NameProvider: "puuid",
	DataDir:      getDefaultDataDir(),
	Orchestrator: types.Orchestrator{
		Enabled:   true,
		Listen:    "0.0.0.0:4222",
		Advertise: "0.0.0.0:4222",
		NodeManager: types.NodeManager{
			DisconnectTimeout: Minute,
		},
		Scheduler: types.Scheduler{
			WorkerCount:          runtime.NumCPU(),
			HousekeepingInterval: 30 * Second,
			HousekeepingTimeout:  2 * Minute,
		},
		EvaluationBroker: types.EvaluationBroker{
			VisibilityTimeout: Minute,
			MaxRetryCount:     10,
		},
	},
	Compute: types.Compute{
		Enabled:       true,
		Orchestrators: []string{"nats://127.0.0.1:4222"},
		Heartbeat: types.Heartbeat{
			InfoUpdateInterval:     Minute,
			ResourceUpdateInterval: 30 * Second,
			Interval:               15 * Second,
		},
		TotalCapacity: types.Resource{
			CPU:    "1",
			Memory: "1Gb",
			Disk:   "1Gb",
			GPU:    "0",
		},
		AllocatedCapacity: types.ResourceScaler{
			CPU:    "80%",
			Memory: "80%",
			Disk:   "80%",
			GPU:    "100%",
		},
	},
	ResultDownloaders: types.ResultDownloaders{
		Timeout: 5 * Minute,
	},
	JobDefaults: types.JobDefaults{
		Batch: types.JobDefaultsConfig{
			Priority: 50,
			Task: types.TaskDefaultConfig{
				Resources: types.Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: types.DefaultPublisherConfig{
					Type: models.PublisherLocal,
				},
				Timeouts: types.TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
		Daemon: types.JobDefaultsConfig{
			Priority: 100,
			Task: types.TaskDefaultConfig{
				Resources: types.Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: types.DefaultPublisherConfig{
					Type: models.PublisherLocal,
				},
				Timeouts: types.TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
		Service: types.JobDefaultsConfig{
			Priority: 50,
			Task: types.TaskDefaultConfig{
				Resources: types.Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: types.DefaultPublisherConfig{
					Type: models.PublisherLocal,
				},
				Timeouts: types.TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
		Ops: types.JobDefaultsConfig{
			Priority: 50,
			Task: types.TaskDefaultConfig{
				Resources: types.Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: types.DefaultPublisherConfig{
					Type: models.PublisherLocal,
				},
				Timeouts: types.TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
	},
	Logging: types.Logging{
		Level:                "INFO",
		Mode:                 "Default",
		LogDebugInfoInterval: 0,
	},
	UpdateConfig: types.UpdateConfig{
		Interval: Day,
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

	log.Error().Msg("Failed to determine default repo path. Using current directory.")
	return defaultBacalhauDir
}
