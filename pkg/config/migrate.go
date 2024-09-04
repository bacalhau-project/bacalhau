package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	v1types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
)

func MigrateV1(in v1types.BacalhauConfig) (types.Bacalhau, error) {
	out := types.Bacalhau{
		API: migrateAPI(in),
		Orchestrator: types.Orchestrator{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "requester"
			}),
			Advertise: in.Node.Network.AdvertisedAddress,
			Cluster:   migrateCluster(in.Node.Network.Cluster),
			NodeManager: types.NodeManager{
				DisconnectTimeout: types.Duration(in.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter),
				ManualApproval:    in.Node.Requester.ManualNodeApproval,
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
			AllowListedLocalPaths: in.Node.AllowListedLocalPaths,
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
		InputSources:      migrateInputSources(in.Node),
		Publishers:        migratePublishers(in.Node),
		Engines:           migrateEngines(in.Node),
		ResultDownloaders: migrateDownloadConfig(in.Node),
		JobAdmissionControl: func(config v1types.NodeConfig) types.JobAdmissionControl {
			out := types.JobAdmissionControl{
				RejectStatelessJobs: in.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
				AcceptNetworkedJobs: in.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
				ProbeHTTP:           in.Node.Requester.JobSelectionPolicy.ProbeHTTP,
				ProbeExec:           in.Node.Requester.JobSelectionPolicy.ProbeExec,
			}

			// compute node configuration takes precedence.
			if in.Node.Compute.JobSelection.RejectStatelessJobs {
				out.RejectStatelessJobs = in.Node.Compute.JobSelection.RejectStatelessJobs
			}
			if in.Node.Compute.JobSelection.AcceptNetworkedJobs {
				out.RejectStatelessJobs = in.Node.Compute.JobSelection.AcceptNetworkedJobs
			}
			if in.Node.Compute.JobSelection.ProbeHTTP != "" {
				out.ProbeHTTP = in.Node.Compute.JobSelection.ProbeHTTP
			}
			if in.Node.Compute.JobSelection.ProbeExec != "" {
				out.ProbeExec = in.Node.Compute.JobSelection.ProbeExec
			}

			return out
		}(in.Node),
		Logging: types.Logging{
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

func migrateCluster(in v1types.NetworkClusterConfig) types.Cluster {
	var out types.Cluster
	if in.Port != 0 {
		out.Port = in.Port
	}
	if len(in.Peers) != 0 {
		out.Peers = in.Peers
	}
	if in.AdvertisedAddress != "" {
		in.AdvertisedAddress = out.Advertise
	}
	return out
}

func migrateAPI(in v1types.BacalhauConfig) types.API {
	var (
		out types.API
	)
	// migrate the existing auth methods
	if len(in.Auth.Methods) > 0 {
		out.Auth.Methods = func(cfg map[string]v1types.AuthenticatorConfig) map[string]types.AuthenticatorConfig {
			out := make(map[string]types.AuthenticatorConfig)
			for k, v := range cfg {
				out[k] = types.AuthenticatorConfig{
					Type:       string(v.Type),
					PolicyPath: v.PolicyPath,
				}
			}
			return out
		}(in.Auth.Methods)
	}
	out.Auth.AccessPolicyPath = in.Auth.AccessPolicyPath
	out.Auth.TokensPath = in.Auth.TokensPath

	// check for a client config, taking lowest precedence.
	if in.Node.ClientAPI.Host != "" {
		out.Host = in.Node.ClientAPI.Host
	}
	if in.Node.ClientAPI.Port != 0 {
		out.Port = in.Node.ClientAPI.Port
	}
	if in.Node.ClientAPI.ClientTLS.UseTLS {
		out.TLS.UseTLS = in.Node.ClientAPI.ClientTLS.UseTLS
	}
	if in.Node.ClientAPI.ClientTLS.CACert != "" {
		out.TLS.CAFile = in.Node.ClientAPI.ClientTLS.CACert
	}
	if in.Node.ClientAPI.ClientTLS.Insecure {
		out.TLS.Insecure = in.Node.ClientAPI.ClientTLS.Insecure
	}

	// check for a server config, taking highest precedence.
	if in.Node.ServerAPI.Host != "" {
		out.Host = in.Node.ServerAPI.Host
	}
	if in.Node.ServerAPI.Port != 0 {
		out.Port = in.Node.ServerAPI.Port
	}
	if in.Node.ServerAPI.TLS.ServerKey != "" {
		out.TLS.KeyFile = in.Node.ServerAPI.TLS.ServerKey
	}
	if in.Node.ServerAPI.TLS.SelfSigned {
		out.TLS.SelfSigned = in.Node.ServerAPI.TLS.SelfSigned
	}
	if in.Node.ServerAPI.TLS.ServerCertificate != "" {
		out.TLS.CertFile = in.Node.ServerAPI.TLS.ServerCertificate
	}
	if in.Node.ServerAPI.TLS.AutoCertCachePath != "" {
		out.TLS.AutoCertCachePath = in.Node.ServerAPI.TLS.AutoCertCachePath
	}
	if in.Node.ServerAPI.TLS.AutoCert != "" {
		out.TLS.AutoCert = in.Node.ServerAPI.TLS.AutoCert
	}

	return out
}

func migrateDownloadConfig(in v1types.NodeConfig) types.ResultDownloaders {
	var out types.ResultDownloaders

	out.Timeout = types.Duration(in.DownloadURLRequestTimeout)
	out.Types.IPFS.Endpoint = in.IPFS.Connect

	return out
}

func migrateEngines(in v1types.NodeConfig) types.EngineConfig {
	var out types.EngineConfig

	// migrate any disabled engines
	out.Disabled = in.DisabledFeatures.Engines

	out.Docker.ManifestCache = types.DockerManifestCache{
		Size:    in.Compute.ManifestCache.Size,
		TTL:     types.Duration(in.Compute.ManifestCache.Duration),
		Refresh: types.Duration(in.Compute.ManifestCache.Frequency),
	}

	return out
}

func migratePublishers(in v1types.NodeConfig) types.PublishersConfig {
	var out types.PublishersConfig

	// migrate any disabled publishers
	out.Disabled = in.DisabledFeatures.Publishers

	// ipfs
	out.Types.IPFS.Endpoint = in.IPFS.Connect

	// local
	out.Types.Local.Port = in.Compute.LocalPublisher.Port
	out.Types.Local.Address = in.Compute.LocalPublisher.Address
	out.Types.Local.Directory = in.Compute.LocalPublisher.Directory

	// s3
	out.Types.S3.PreSignedURLExpiration = types.Duration(in.Requester.StorageProvider.S3.PreSignedURLExpiration)
	out.Types.S3.PreSignedURLDisabled = in.Requester.StorageProvider.S3.PreSignedURLDisabled

	return out
}

func migrateInputSources(in v1types.NodeConfig) types.InputSourcesConfig {
	var out types.InputSourcesConfig

	// migrate any disabled storages
	out.Disabled = in.DisabledFeatures.Storages

	// ipfs
	out.Types.IPFS.Endpoint = in.IPFS.Connect

	return out
}
