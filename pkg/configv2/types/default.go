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
	API: API{
		Address: "http://0.0.0.0:1234",
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
		Enabled:   true,
		Listen:    "0.0.0.0:4222",
		Advertise: "0.0.0.0:4222",
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
		Enabled:       true,
		Orchestrators: []string{"nats://127.0.0.1:4222"},
		Heartbeat: Heartbeat{
			InfoUpdateInterval:     Minute,
			ResourceUpdateInterval: 30 * Second,
			Interval:               15 * Second,
		},
		TotalCapacity: Resource{
			CPU:    "1",
			Memory: "1Gb",
			Disk:   "1Gb",
			GPU:    "0",
		},
		AllocatedCapacity: ResourceScaler{
			CPU:    "80%",
			Memory: "80%",
			Disk:   "80%",
			GPU:    "100%",
		},
	},
	ResultDownloaders: ResultDownloaders{
		Timeout: 5 * Minute,
	},
	JobDefaults: JobDefaults{
		Batch: JobDefaultsConfig{
			Priority: 50,
			Task: TaskDefaultConfig{
				Resources: Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: DefaultPublisherConfig{
					Type: KindPublisherLocal,
				},
				Timeouts: TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
		Daemon: JobDefaultsConfig{
			Priority: 100,
			Task: TaskDefaultConfig{
				Resources: Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: DefaultPublisherConfig{
					Type: KindPublisherLocal,
				},
				Timeouts: TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
		Service: JobDefaultsConfig{
			Priority: 50,
			Task: TaskDefaultConfig{
				Resources: Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: DefaultPublisherConfig{
					Type: KindPublisherLocal,
				},
				Timeouts: TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
		Ops: JobDefaultsConfig{
			Priority: 50,
			Task: TaskDefaultConfig{
				Resources: Resource{
					CPU:    "500m",
					Memory: "1Gb",
				},
				Publisher: DefaultPublisherConfig{
					Type: KindPublisherLocal,
				},
				Timeouts: TaskTimeoutConfig{
					ExecutionTimeout: 30 * Minute,
				},
			},
		},
	},
	Logging: Logging{
		Level:                "INFO",
		Mode:                 "Default",
		LogDebugInfoInterval: 0,
	},
	UpdateConfig: UpdateConfig{
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

	return defaultBacalhauDir
}
