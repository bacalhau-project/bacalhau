package configv2

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"

	v1types "github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func MigrateV1(in v1types.BacalhauConfig) (types.Bacalhau, error) {
	publisherConfig, err := migratePublishers(in.Node)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("migrating publisher config: %w", err)
	}
	executorConfig, err := migrateEngines(in.Node)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("migrating executor config: %w", err)
	}
	out := types.Bacalhau{
		DataDir: "~/.bacalhau",
		// TODO(forrest) [review]: when migrating should the address come from the server or client when both are present?
		API: types.API{
			Address: "TODO",
			TLS: types.TLS{
				// TODO(forrest) [review]: when migrating if both the server and client have TLS configs, which do we take?
				CertFile: "TODO",
				KeyFile:  "TODO",
				CAFile:   "TODO",
			},
			Auth: types.AuthConfig{
				TokensPath: in.Auth.TokensPath,
				Methods: func(cfg map[string]v1types.AuthenticatorConfig) map[string]types.AuthenticatorConfig {
					out := make(map[string]types.AuthenticatorConfig)
					for k, v := range cfg {
						out[k] = types.AuthenticatorConfig{
							Type:       string(v.Type),
							PolicyPath: v.PolicyPath,
						}
					}
					return out
				}(in.Auth.Methods),
				AccessPolicyPath: in.Auth.AccessPolicyPath,
			},
		},
		Orchestrator: types.Orchestrator{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "requester"
			}),
			Advertise: in.Node.Network.AdvertisedAddress,
			Cluster: types.Cluster{
				Listen:    fmt.Sprintf("0.0.0.0:%d", in.Node.Network.Cluster.Port),
				Advertise: in.Node.Network.Cluster.AdvertisedAddress,
				Peers:     in.Node.Network.Cluster.Peers,
			},
			NodeManager: types.NodeManager{
				DisconnectTimeout: types.Duration(in.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter),
				ManualApproval:    !in.Node.Requester.ManualNodeApproval,
			},
			Scheduler: types.Scheduler{
				WorkerCount:          in.Node.Requester.Worker.WorkerCount,
				HousekeepingInterval: types.Duration(in.Node.Requester.HousekeepingBackgroundTaskInterval),
			},
			EvaluationBroker: types.EvaluationBroker{
				VisibilityTimeout: types.Duration(in.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout),
				MaxRetryCount:     in.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount,
			},
		},
		Compute: types.Compute{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "compute"
			}),
			Orchestrators: in.Node.Network.Orchestrators,
			Labels:        in.Node.Labels,
			Heartbeat: types.Heartbeat{
				Interval:               types.Duration(in.Node.Compute.ControlPlaneSettings.HeartbeatFrequency),
				ResourceUpdateInterval: types.Duration(in.Node.Compute.ControlPlaneSettings.ResourceUpdateFrequency),
				InfoUpdateInterval:     types.Duration(in.Node.Compute.ControlPlaneSettings.InfoUpdateFrequency),
			},
			TotalCapacity: types.Resource{
				CPU:    in.Node.Compute.Capacity.TotalResourceLimits.CPU,
				Memory: in.Node.Compute.Capacity.TotalResourceLimits.Memory,
				Disk:   in.Node.Compute.Capacity.TotalResourceLimits.Disk,
				GPU:    in.Node.Compute.Capacity.TotalResourceLimits.GPU,
			},
			Volumes: func(paths []string) []types.Volume {
				out := make([]types.Volume, len(paths))
				for i, p := range paths {
					out[i] = types.Volume{
						Name: p,
						Path: p,
					}
				}
				return out
			}(in.Node.AllowListedLocalPaths),
		},
		WebUI: types.WebUI{
			Enabled: in.Node.WebUI.Enabled,
			Listen: func(enabled bool, port int) string {
				if enabled {
					return fmt.Sprintf("0.0.0.0:%d", port)
				}
				return ""
			}(in.Node.WebUI.Enabled, in.Node.WebUI.Port),
		},
		InputSources: types.InputSourcesConfig{
			Disabled: in.Node.DisabledFeatures.Storages,
			Config:   migrateInputSources(in.Node),
		},
		Publishers: types.PublishersConfig{
			Disabled: in.Node.DisabledFeatures.Publishers,
			Config:   publisherConfig,
		},
		Executors: types.ExecutorsConfig{
			Disabled: in.Node.DisabledFeatures.Engines,
			Config:   executorConfig,
		},
		ResultDownloaders: types.ResultDownloaders{
			// TODO(forrest) [review]: unsure what value we should include here, if any.
			Timeout: types.Duration(in.Node.DownloadURLRequestTimeout),
			Config:  migrateDownloadConfig(in.Node),
		},
		// TODO(forrest) [review] the decision regarding the presence of these values or if we even use them has
		// not been made https://www.notion.so/Rethinking-Configuration-435fbe87419148b4bbc5119d413786eb?d=1c73cb661b18405b8b67b3856f02e1d1&pvs=4#fdc6de7c00364de1bb81b325686e6e66
		// It sounds like we only want to allow the setting of a default publisher, but unsure if this is on a job
		// type basis or something else
		JobDefaults: types.JobDefaults{
			Batch: types.JobDefaultsConfig{
				// TODO(forrest) [review]: Does a zero publisher value imply it has no priority or that it's not configured?
				// Priority: 100,
				Task: types.TaskDefaultConfig{
					Resources: types.Resource{
						CPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.CPU,
						Memory: in.Node.Compute.Capacity.DefaultJobResourceLimits.Memory,
						Disk:   in.Node.Compute.Capacity.DefaultJobResourceLimits.Disk,
						GPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.GPU,
					},
					Publisher: types.DefaultPublisherConfig{
						Type: in.Node.Requester.DefaultPublisher,
					},
					Timeouts: types.TaskTimeoutConfig{
						ExecutionTimeout: types.Duration(in.Node.Compute.JobTimeouts.MaxJobExecutionTimeout),
					},
				},
			},
			Daemon: types.JobDefaultsConfig{
				// TODO(forrest) [review]: Does a zero publisher value imply it has no priority or that it's not configured?
				// Priority: 100,
				Task: types.TaskDefaultConfig{
					Resources: types.Resource{
						CPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.CPU,
						Memory: in.Node.Compute.Capacity.DefaultJobResourceLimits.Memory,
						Disk:   in.Node.Compute.Capacity.DefaultJobResourceLimits.Disk,
						GPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.GPU,
					},
					Publisher: types.DefaultPublisherConfig{
						Type: in.Node.Requester.DefaultPublisher,
					},
					Timeouts: types.TaskTimeoutConfig{
						ExecutionTimeout: types.Duration(in.Node.Compute.JobTimeouts.MaxJobExecutionTimeout),
					},
				},
			},
			Service: types.JobDefaultsConfig{
				// TODO(forrest) [review]: Does a zero publisher value imply it has no priority or that it's not configured?
				// Priority: 100,
				Task: types.TaskDefaultConfig{
					Resources: types.Resource{
						CPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.CPU,
						Memory: in.Node.Compute.Capacity.DefaultJobResourceLimits.Memory,
						Disk:   in.Node.Compute.Capacity.DefaultJobResourceLimits.Disk,
						GPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.GPU,
					},
					Publisher: types.DefaultPublisherConfig{
						Type: in.Node.Requester.DefaultPublisher,
					},
					Timeouts: types.TaskTimeoutConfig{
						ExecutionTimeout: types.Duration(in.Node.Compute.JobTimeouts.MaxJobExecutionTimeout),
					},
				},
			},
			Ops: types.JobDefaultsConfig{
				// TODO(forrest) [review]: Does a zero publisher value imply it has no priority or that it's not configured?
				// Priority: 100,
				Task: types.TaskDefaultConfig{
					Resources: types.Resource{
						CPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.CPU,
						Memory: in.Node.Compute.Capacity.DefaultJobResourceLimits.Memory,
						Disk:   in.Node.Compute.Capacity.DefaultJobResourceLimits.Disk,
						GPU:    in.Node.Compute.Capacity.DefaultJobResourceLimits.GPU,
					},
					Publisher: types.DefaultPublisherConfig{
						Type: in.Node.Requester.DefaultPublisher,
					},
					Timeouts: types.TaskTimeoutConfig{
						ExecutionTimeout: types.Duration(in.Node.Compute.JobTimeouts.MaxJobExecutionTimeout),
					},
				},
			},
		},
		// TODO(forrest) [review]: currently both the compute and requester have a job selection policy
		// it is not clear whose policy should be migrated here.
		JobAdmissionControl: types.JobAdmissionControl{
			RejectStatelessJobs: in.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
			AcceptNetworkedJobs: in.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
			ProbeHTTP:           in.Node.Requester.JobSelectionPolicy.ProbeHTTP,
			ProbeExec:           in.Node.Requester.JobSelectionPolicy.ProbeExec,
		},
		Logging: types.Logging{
			Level:                "info",
			Mode:                 string(in.Node.LoggingMode),
			LogDebugInfoInterval: types.Duration(in.Node.Compute.Logging.LogRunningExecutionsInterval),
		},
		UpdateConfig: types.UpdateConfig{
			Interval: types.Duration(in.Update.CheckFrequency),
		},
		FeatureFlags: types.FeatureFlags{
			ExecTranslation: in.Node.Requester.TranslationEnabled,
		},
	}
	return out, nil
}

func migrateDownloadConfig(in v1types.NodeConfig) map[string]map[string]string {
	out := make(map[string]map[string]string)
	if in.IPFS.Connect != "" {
		// TODO(forrest) [review]: unsure what name to key here
		out[models.StorageSourceIPFS] = map[string]string{
			"endpoint": in.IPFS.Connect,
		}
	}
	// TODO(forrest) [review]: assuredly the presence of these keys in the map indicate
	// they are enable. I think we want to enable these by default, but at the same time
	// we don't want to use, for example, the S3 downloader if it's not configured correctly.
	// TODO(forrest) [review]: unsure what name to key here
	out[models.StorageSourceS3] = make(map[string]string)
	// TODO(forrest) [review]: unsure what name to key here
	out[models.StorageSourceURL] = make(map[string]string)

	return out
}

func migrateEngines(in v1types.NodeConfig) (map[string]map[string]interface{}, error) {
	out := make(map[string]map[string]interface{})
	if !slices.ContainsFunc(in.DisabledFeatures.Engines, func(s string) bool {
		return strings.ToLower(s) == models.EngineDocker
	}) {
		config := make(map[string]interface{})
		var cacheConfig map[string]interface{}
		if err := mapstructure.Decode(in.Compute.ManifestCache, &cacheConfig); err != nil {
			return nil, err
		}
		config["manifestcache"] = cacheConfig
		out[models.EngineDocker] = config
	}
	if !slices.ContainsFunc(in.DisabledFeatures.Engines, func(s string) bool {
		return strings.ToLower(s) == models.EngineWasm
	}) {
		config := make(map[string]interface{})
		out[models.EngineWasm] = config
	}
	return out, nil
}

func migratePublishers(in v1types.NodeConfig) (map[string]map[string]interface{}, error) {
	out := make(map[string]map[string]interface{})
	if !slices.ContainsFunc(in.DisabledFeatures.Publishers, func(s string) bool {
		return strings.ToLower(s) == models.PublisherIPFS
	}) && in.IPFS.Connect != "" {
		config := make(map[string]interface{})
		config["endpoint"] = in.IPFS.Connect
		out[models.PublisherIPFS] = config
	}
	if !slices.ContainsFunc(in.DisabledFeatures.Publishers, func(s string) bool {
		return strings.ToLower(s) == models.PublisherS3
	}) {
		// TODO(forrest) [review]: should we attempt to extract values in the environment for this?
		// should we include the PreSignedURL bits from the v1 config?
		out[models.PublisherS3] = make(map[string]interface{})
	}
	if !slices.ContainsFunc(in.DisabledFeatures.Publishers, func(s string) bool {
		return strings.ToLower(s) == models.PublisherLocal
	}) {
		var config map[string]interface{}
		if err := mapstructure.Decode(in.Compute.LocalPublisher, &config); err != nil {
			return nil, err
		}
		out[models.PublisherLocal] = config
	}
	return out, nil
}

// TODO(forrest) [review]: other storage sources to consider here include:
//
//	S3PreSigned
//	Inline
func migrateInputSources(in v1types.NodeConfig) map[string]map[string]interface{} {
	out := make(map[string]map[string]interface{})
	// if the URL input source isn't listed as disabled, enable it
	if !slices.ContainsFunc(in.DisabledFeatures.Storages, func(s string) bool {
		return strings.ToLower(s) == models.StorageSourceURL
	}) {
		// there isn't configuration for this storage, meaning its presents in the map implies it's enabled.
		out[models.StorageSourceURL] = make(map[string]interface{})
	}

	// if the IPFS input source isn't listed as disabled, and it has a config value, enable it
	if !slices.ContainsFunc(in.DisabledFeatures.Storages, func(s string) bool {
		return strings.ToLower(s) == models.StorageSourceIPFS
	}) && in.IPFS.Connect != "" {
		config := make(map[string]interface{})
		config["endpoint"] = in.IPFS.Connect
		out[models.StorageSourceIPFS] = config
	}

	if !slices.ContainsFunc(in.DisabledFeatures.Storages, func(s string) bool {
		return strings.ToLower(s) == models.StorageSourceLocalDirectory
	}) {
		config := make(map[string]interface{})
		// TODO(forrest) [review] unsure how to configure this.
		// we are mixing configuration across fields here, Volumes on the compute node should be what determine this
		// but that isn't part of this config structure, so configure it in both spots I guess?
		config["volumes"] = in.AllowListedLocalPaths
		out[models.StorageSourceLocalDirectory] = config
	}
	if !slices.ContainsFunc(in.DisabledFeatures.Storages, func(s string) bool {
		return strings.ToLower(s) == models.StorageSourceS3
	}) {
		// TODO(forrest) [review]: should we attempt to extract values in the environment for this?
		out[models.StorageSourceS3] = make(map[string]interface{})
	}

	return out
}
