package configv2

import (
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var Default = types.Bacalhau{
	Repo: "~/.bacalhau",
	Name: "",
	Client: types.Client{
		Address: "http://0.0.0.0:1234",
	},
	Server: types.Server{
		Address: "0.0.0.0",
		Port:    1234,
		TLS:     types.TLS{},
		Auth: types.AuthConfig{
			Methods: map[string]types.AuthenticatorConfig{
				"ClientKey": {
					Type: "challenge",
				},
			},
		},
	},
	Orchestrator: types.Orchestrator{
		Enabled:   true,
		Listen:    "0.0.0.0",
		Port:      4222,
		Advertise: "0.0.0.0",
		NodeManager: types.NodeManager{
			DisconnectTimeout: types.Duration(time.Second * 30),
			AutoApprove:       true,
		},
		Scheduler: types.Scheduler{
			Workers:              runtime.NumCPU(),
			HousekeepingInterval: types.Duration(time.Second * 30),
			HousekeepingTimeout:  types.Duration(time.Minute * 2),
		},
		Broker: types.EvaluationBroker{
			VisibilityTimeout: types.Duration(time.Minute),
			MaxRetries:        10,
		},
	},
	Compute: types.Compute{
		Enabled:       true,
		Orchestrators: []string{"0.0.0.0:4222"},
		Heartbeat: types.Heartbeat{
			MessageInterval:  types.Duration(time.Second * 30),
			ResourceInterval: types.Duration(time.Second * 30),
			InfoInterval:     types.Duration(time.Second * 30),
		},
		Capacity: types.Capacity{
			Allocated: types.ResourceScaler{
				CPU:    "80%",
				Memory: "80%",
				Disk:   "80%",
				GPU:    "100%",
			},
		},
		Storages: types.Storage{
			HTTP: types.HTTPStorage{
				Enabled: true,
			},
		},
		Engines: types.Engine{
			Docker: types.Docker{
				Enabled: true,
				ManifestCache: types.DockerManifestCache{
					Size:    5 << 30, // 5GB
					TTL:     types.Duration(time.Hour * 24),
					Refresh: types.Duration(time.Hour * 24),
				},
			},
			WASM: types.WASM{
				Enabled: true,
			},
		},
		Policy: types.SelectionPolicy{
			Networked: true,
			Local:     false,
		},
	},
	Telemetry: types.Telemetry{
		Logging: types.Logging{
			Level:  "info",
			Format: "console",
		},
	},
}
