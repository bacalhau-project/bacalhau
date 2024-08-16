package configv2

import (
	"fmt"
	"slices"
	"strings"

	v1types "github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func MigrateV1(in v1types.BacalhauConfig) (types.Bacalhau, error) {
	out := types.Bacalhau{
		Repo: "",
		Name: in.Node.Name,
		Client: types.Client{
			Address:     fmt.Sprintf("http://%s:%d", in.Node.ClientAPI.Host, in.Node.ClientAPI.Port),
			Certificate: in.Node.ClientAPI.ClientTLS.CACert,
			Insecure:    in.Node.ClientAPI.ClientTLS.Insecure,
		},
		Server: types.Server{
			Address: in.Node.ServerAPI.Host,
			Port:    in.Node.ServerAPI.Port,
			TLS: types.TLS{
				AutoCert:          in.Node.ServerAPI.TLS.AutoCert,
				AutoCertCachePath: in.Node.ServerAPI.TLS.AutoCertCachePath,
				Certificate:       in.Node.ServerAPI.TLS.ServerCertificate,
				Key:               in.Node.ServerAPI.TLS.ServerKey,
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
			Port:       in.Node.Network.Port,
			Advertise:  in.Node.Network.AdvertisedAddress,
			AuthSecret: in.Node.Network.AuthSecret,
			Cluster: types.Cluster{
				Name:      in.Node.Network.Cluster.Name,
				Port:      in.Node.Network.Cluster.Port,
				Advertise: in.Node.Network.Cluster.AdvertisedAddress,
				Peers:     in.Node.Network.Cluster.Peers,
			},
			NodeManager: types.NodeManager{
				DisconnectTimeout: types.Duration(in.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter),
				AutoApprove:       !in.Node.Requester.ManualNodeApproval,
			},
			Scheduler: types.Scheduler{
				Workers:              in.Node.Requester.Worker.WorkerCount,
				HousekeepingInterval: types.Duration(in.Node.Requester.HousekeepingBackgroundTaskInterval),
			},
			Broker: types.EvaluationBroker{
				VisibilityTimeout: types.Duration(in.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout),
				MaxRetries:        in.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount,
			},
		},
		Compute: types.Compute{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "compute"
			}),
			Orchestrators: in.Node.Network.Orchestrators,
			Labels:        in.Node.Labels,
			Heartbeat: types.Heartbeat{
				MessageInterval:  types.Duration(in.Node.Compute.ControlPlaneSettings.HeartbeatFrequency),
				ResourceInterval: types.Duration(in.Node.Compute.ControlPlaneSettings.ResourceUpdateFrequency),
				InfoInterval:     types.Duration(in.Node.Compute.ControlPlaneSettings.InfoUpdateFrequency),
			},
			Capacity: types.Capacity{
				Total: types.Resource{
					CPU:    in.Node.Compute.Capacity.TotalResourceLimits.CPU,
					Memory: in.Node.Compute.Capacity.TotalResourceLimits.Memory,
					Disk:   in.Node.Compute.Capacity.TotalResourceLimits.Disk,
					GPU:    in.Node.Compute.Capacity.TotalResourceLimits.GPU,
				},
			},
			Publishers: types.Publisher{
				IPFS: types.IPFSPublisher{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Publishers, func(s string) bool {
						return strings.ToLower(s) == models.PublisherIPFS
					}),
					Endpoint: in.Node.IPFS.Connect,
				},
				// NB(forrest): this is challenging as there was never a config file this, it lives in environment variables...
				// it's also mixing the responsibility of compute and requester, since the requester used the Pre-signed bits..
				S3: types.S3Publisher{
					Enabled:   false,
					Endpoint:  "",
					AccessKey: "",
					SecretKey: "",
					// TODO(forrest) [review] unsure if we should even be migrating these here..
					PreSignedURLEnabled:    !in.Node.Requester.StorageProvider.S3.PreSignedURLDisabled,
					PreSignedURLExpiration: types.Duration(in.Node.Requester.StorageProvider.S3.PreSignedURLExpiration),
				},
				LocalHTTPServer: types.LocalHTTPServerPublisher{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Publishers, func(s string) bool {
						return strings.ToLower(s) == models.PublisherLocal
					}),
					Host: in.Node.Compute.LocalPublisher.Address,
					Port: in.Node.Compute.LocalPublisher.Port,
				},
			},
			Storages: types.Storage{
				HTTP: types.HTTPStorage{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Storages, func(s string) bool {
						return strings.ToLower(s) == models.StorageSourceURL
					}),
				},
				IPFS: types.IPFSStorage{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Storages, func(s string) bool {
						return strings.ToLower(s) == models.StorageSourceIPFS
					}),
					Endpoint: in.Node.IPFS.Connect,
				},
				Local: types.LocalStorage{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Storages, func(s string) bool {
						return strings.ToLower(s) == models.StorageSourceLocalDirectory
					}),
					Volumes: func(paths []string) []types.Volume {
						out := make([]types.Volume, len(paths))
						for i, p := range paths {
							out[i] = types.Volume{
								Name:  p,
								Path:  p,
								Write: false,
							}
						}
						return out
					}(in.Node.AllowListedLocalPaths),
				},
				// NB(forrest): this is challenging as there was never a config file this, it lives in environment variables...
				// for now mark it as disabled
				S3: types.S3Storage{
					Enabled: false,
				},
			},
			Engines: types.Engine{
				Docker: types.Docker{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Engines, func(s string) bool {
						return strings.ToLower(s) == models.EngineDocker
					}),
					ManifestCache: types.DockerManifestCache{
						Size:    in.Node.Compute.ManifestCache.Size,
						TTL:     types.Duration(in.Node.Compute.ManifestCache.Duration),
						Refresh: types.Duration(in.Node.Compute.ManifestCache.Frequency),
					},
				},
				WASM: types.WASM{
					Enabled: !slices.ContainsFunc(in.Node.DisabledFeatures.Engines, func(s string) bool {
						return strings.ToLower(s) == models.EngineWasm
					}),
				},
			},
			Policy: types.SelectionPolicy{
				Networked: in.Node.Compute.JobSelection.AcceptNetworkedJobs,
				Local:     in.Node.Compute.JobSelection.Locality == models.Local,
			},
		},
		Telemetry: types.Telemetry{
			Logging: types.Logging{
				Level: "info",
				Format: func(mode logger.LogMode) string {
					if mode == logger.LogModeJSON {
						return "json"
					}
					return "console"
				}(in.Node.LoggingMode),
			},
		},
	}
	return out, nil
}
