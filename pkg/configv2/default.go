package configv2

import (
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	Day    = types.Duration(time.Hour * 24)
	Hour   = types.Duration(time.Hour)
	Minuet = types.Duration(time.Minute)
	Second = types.Duration(time.Second)
)

// NB(forrest): setting copied verbatim from:
// https://www.notion.so/expanso/Rethinking-Configuration-435fbe87419148b4bbc5119d413786eb?pvs=4#6a28290e0c514e3b95e8ec6ee0106379

// Default is the default configuration for a bacalhau node.
var Default = types.Bacalhau{
	API: types.API{
		Address: "0.0.0.0:1234",
	},
	DataDir:            "~/.bacalhau",
	NameProvider:       "puuid",
	StrictVersionMatch: false,
	Orchestrator: types.Orchestrator{
		Enabled: true,
		Listen:  "0.0.0.0:4222",
		Cluster: types.Cluster{
			Listen: "0.0.0.0:4222",
		},
		NodeManager: types.NodeManager{
			GCThreshold:       Day,
			GCInterval:        10 * Minuet,
			DisconnectTimeout: Minuet,
			ManualApproval:    false,
		},
		StateStore: types.OrchestratorStateStore{
			JobGCInterval:   10 * Minuet,
			JobGCThreshold:  Day,
			EvalGCThreshold: Hour,
			Backend: types.StoreBackend{
				Type:   "BoltDB",
				Config: make(map[string]string),
			},
		},
		Scheduler: types.Scheduler{
			WorkerCount:          runtime.NumCPU(),
			HousekeepingInterval: 30 * Second,
			HousekeepingTimeout:  2 * Minuet,
		},
		EvaluationBroker: types.EvaluationBroker{
			VisibilityTimeout: Minuet,
			MaxRetryCount:     10,
		},
	},
	Compute: types.Compute{
		Enabled: true,
		Heartbeat: types.Heartbeat{
			InfoUpdateInterval:     Minuet,
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
		StateStore: types.ComputeStateStore{
			ExecutionGCInterval:  10 * Minuet,
			ExecutionGCThreshold: Day,
			Backend: types.StoreBackend{
				Type:   "BoltDB",
				Config: make(map[string]string),
			},
		},
	},
	WebUI: types.WebUI{
		Enabled: false,
		Listen:  "0.0.0.0:8483",
	},
	InputSources: types.InputSourcesConfig{
		ReadTimeout: 5 * Minuet,
		MazSize:     "100GB",
		Decompression: types.DecompressionConfig{
			SizeLimit:      "100GB",
			FileCountLimit: 1000,
		},
		Config: map[string]map[string]interface{}{
			models.StorageSourceURL: make(map[string]interface{}),
			// TODO(forrest) [review]: we need this here to support running wasm jobs
			// as the wasm binary is inlined
			models.StorageSourceInline: make(map[string]interface{}),
		},
	},
	Publishers: types.PublishersConfig{
		Config: map[string]map[string]interface{}{
			models.PublisherLocal: {
				// TODO(forrest) [review]: this was the default, but public isn't a valid host. It could be worth changing.
				"address": "public",
				"port":    6001,
			},
		},
	},
	Executors: types.ExecutorsConfig{
		Config: map[string]map[string]interface{}{
			models.EngineDocker: {
				"manifestcache": map[string]interface{}{
					"size":             1000,
					"duration":         Hour,
					"refreshFrequency": Hour,
				},
			},
			models.EngineWasm: make(map[string]interface{}),
		},
	},
	ResultDownloaders: types.ResultDownloaders{
		Timeout: 5 * Minuet,
		Config: map[string]map[string]string{
			models.StorageSourceS3:  make(map[string]string),
			models.StorageSourceURL: make(map[string]string),
		},
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
					ExecutionTimeout: 30 * Minuet,
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
					ExecutionTimeout: 30 * Minuet,
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
					ExecutionTimeout: 30 * Minuet,
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
					ExecutionTimeout: 30 * Minuet,
				},
			},
		},
	},
	JobAdmissionControl: types.JobAdmissionControl{
		RejectStatelessJobs: false,
		AcceptNetworkedJobs: false,
		ProbeHTTP:           "",
		ProbeExec:           "",
	},
	Logging: types.Logging{
		Level:                "INFO",
		Mode:                 "Default",
		LogDebugInfoInterval: 0,
	},
	UpdateConfig: types.UpdateConfig{
		Interval: Day,
	},
	FeatureFlags: types.FeatureFlags{
		ExecTranslation: false,
	},
}
