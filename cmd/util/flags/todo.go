package flags

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type FlagDefinition struct {
	FlagName     string
	ConfigPath   string
	DefaultValue interface{}
	Description  string
}

// Generic function to define a flag, set a default value, and bind it to Viper
func RegisterFlags(cmd *cobra.Command, register map[string][]FlagDefinition) error {
	for name, defs := range register {
		fset := pflag.NewFlagSet(name, pflag.ContinueOnError)
		// Determine the type of the default value
		for _, def := range defs {
			switch v := def.DefaultValue.(type) {
			case int:
				fset.Int(def.FlagName, v, def.Description)
			case bool:
				fset.Bool(def.FlagName, v, def.Description)
			case string:
				fset.String(def.FlagName, v, def.Description)
			case []string:
				fset.StringSlice(def.FlagName, v, def.Description)
			case map[string]string:
				fset.StringToString(def.FlagName, v, def.Description)
			case model.JobSelectionDataLocality:
				fset.Var(DataLocalityFlag(&v), def.FlagName, def.Description)
			default:
				return fmt.Errorf("unhandled type: %T", v)
			}
			viper.SetDefault(def.ConfigPath, def.DefaultValue)
			if err := viper.BindPFlag(def.ConfigPath, fset.Lookup(def.FlagName)); err != nil {
				return err
			}
		}
		cmd.PersistentFlags().AddFlagSet(fset)
	}
	return nil
}

const (
	// Constants for DisabledFeatureFlags
	NodeDisabledFeatureEngines    = "Node.DisabledFeature.Engines"
	NodeDisabledFeaturePublishers = "Node.DisabledFeature.Publishers"
	NodeDisabledFeatureStorages   = "Node.DisabledFeature.Storages"

	// Constants for CapacityFlags
	NodeComputeCapacityClientIDBypass = "Node.Compute.Capacity.ClientIDBypass"
	NodeComputeCapacityTotalCPU       = "Node.Compute.Capacity.TotalCPU"
	NodeComputeCapacityTotalMemory    = "Node.Compute.Capacity.TotalMemory"
	NodeComputeCapacityTotalGPU       = "Node.Compute.Capacity.TotalGPU"
	NodeComputeCapacityJobCPU         = "Node.Compute.Capacity.JobCPU"
	NodeComputeCapacityJobMemory      = "Node.Compute.Capacity.JobMemory"
	NodeComputeCapacityJobGPU         = "Node.Compute.Capacity.JobGPU"

	// Constants for EstuaryFlags
	NodeEstuaryAPIKey = "Node.EstuaryAPIKey"

	// Constants for IPFSFlags
	NodeIPFSSwarmAddress    = "Node.IPFS.SwarmAddress"
	NodeIPFSConnect         = "Node.IPFS.Connect"
	NodeIPFSPrivateInternal = "Node.IPFS.PrivateInternal"

	// Constants for JobSelectionFlags
	NodeRequesterJobSelectionPolicyLocality            = "Node.Requester.JobSelectionPolicy.Locality"
	NodeRequesterJobSelectionPolicyRejectStatelessJobs = "Node.Requester.JobSelectionPolicy.RejectStatelessJobs"
	NodeRequesterJobSelectionPolicyAcceptNetworkedJobs = "Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs"
	NodeRequesterJobSelectionPolicyProbeHTTP           = "Node.Requester.JobSelectionPolicy.ProbeHTTP"
	NodeRequesterJobSelectionPolicyProbeExec           = "Node.Requester.JobSelectionPolicy.ProbeExec"

	// Constants for LabelFlags
	NodeLabels = "Node.Labels"

	// Constants for Libp2pFlags
	NodeLibp2pPeerConnect = "Node.Libp2p.PeerConnect"
	NodeLibp2pSwarmPort   = "Node.Libp2p.SwarmPort"

	// Constants for AllowListLocalPathsFlags
	NodeAllowListedLocalPaths = "Node.AllowListedLocalPaths"

	// Constants for NodeTypeFlags
	NodeType = "Node.Type"
)

func Register() error {
	return RegisterFlags(nil, DefaultFlagConfig())
}

func DefaultFlagConfig() map[string][]FlagDefinition {
	return map[string][]FlagDefinition{
		"disabled-features": []FlagDefinition{
			{
				FlagName:     "disable-engine",
				ConfigPath:   NodeDisabledFeatureEngines,
				DefaultValue: config_v2.Default.Node.DisabledFeatures.Engines,
				Description:  "Engine types to disable",
			},
			{
				FlagName:     NodeDisabledFeaturePublishers,
				ConfigPath:   "Node.DisabledFeature.Publishers",
				DefaultValue: config_v2.Default.Node.DisabledFeatures.Publishers,
				Description:  "Engine types to disable",
			},
			{
				FlagName:     "disable-storage",
				ConfigPath:   NodeDisabledFeatureStorages,
				DefaultValue: config_v2.Default.Node.DisabledFeatures.Storages,
				Description:  "Engine types to disable",
			},
		},
		"capacity": []FlagDefinition{
			{
				FlagName:     "job-execution-timeout-bypass-client-id",
				ConfigPath:   NodeComputeCapacityClientIDBypass,
				DefaultValue: config_v2.Default.Node.Compute.ClientIDBypass,
				Description:  `List of IDs of clients that are allowed to bypass the job execution timeout check`,
			},
			{
				FlagName:     "limit-total-cpu",
				ConfigPath:   NodeComputeCapacityTotalCPU,
				DefaultValue: config_v2.Default.Node.Compute.Capacity.TotalCPU,
				Description:  `Total CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
			},
			{
				FlagName:     "limit-total-memory",
				ConfigPath:   NodeComputeCapacityTotalMemory,
				DefaultValue: config_v2.Default.Node.Compute.Capacity.TotalMemory,
				Description:  `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
			},
			{
				FlagName:     "limit-total-gpu",
				ConfigPath:   NodeComputeCapacityTotalGPU,
				DefaultValue: config_v2.Default.Node.Compute.Capacity.TotalGPU,
				Description:  `Total GPU limit to run all jobs (e.g. 1, 2, or 8).`,
			},
			{
				FlagName:     "limit-job-cpu",
				ConfigPath:   NodeComputeCapacityJobCPU,
				DefaultValue: config_v2.Default.Node.Compute.Capacity.JobCPU,
				Description:  `Job CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
			},
			{
				FlagName:     "limit-job-memory",
				ConfigPath:   NodeComputeCapacityJobMemory,
				DefaultValue: config_v2.Default.Node.Compute.Capacity.JobMemory,
				Description:  `Job Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
			},
			{
				FlagName:     "limit-job-gpu",
				ConfigPath:   NodeComputeCapacityJobGPU,
				DefaultValue: config_v2.Default.Node.Compute.Capacity.JobGPU,
				Description:  `Job GPU limit to run all jobs (e.g. 1, 2, or 8).`,
			},
		},
		"estuary": []FlagDefinition{
			{
				FlagName:     "estuary-api-key",
				ConfigPath:   NodeEstuaryAPIKey,
				DefaultValue: config_v2.Default.Node.EstuaryAPIKey,
				Description:  `The API key used when using the estuary API.`,
			},
		},
		"ipfs": []FlagDefinition{
			{
				FlagName:     "ipfs-swarm-addr",
				ConfigPath:   NodeIPFSSwarmAddress,
				DefaultValue: config_v2.Default.Node.IPFS.SwarmAddresses,
				Description:  "IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.",
			},
			{
				FlagName:     "ipfs-connect",
				ConfigPath:   NodeIPFSConnect,
				DefaultValue: config_v2.Default.Node.IPFS.Connect,
				Description:  "The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.",
			},
			{
				FlagName:     "private-internal-ipfs",
				ConfigPath:   NodeIPFSPrivateInternal,
				DefaultValue: config_v2.Default.Node.IPFS.PrivateInternal,
				Description: "Whether the in-process IPFS node should auto-discover other nodes, including the public IPFS network - " +
					"cannot be used with --ipfs-connect. " +
					"Use \"--private-internal-ipfs=false\" to disable. " +
					"To persist a local Ipfs node, set BACALHAU_SERVE_IPFS_PATH to a valid path.",
			},
		},
		"job-selection": []FlagDefinition{
			{
				FlagName:     "job-selection-data-locality",
				ConfigPath:   NodeRequesterJobSelectionPolicyLocality,
				DefaultValue: config_v2.Default.Node.Requester.JobSelectionPolicy.Locality,
				Description:  `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
			},
			{
				FlagName:     "job-selection-reject-stateless",
				ConfigPath:   NodeRequesterJobSelectionPolicyRejectStatelessJobs,
				DefaultValue: config_v2.Default.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
				Description:  `Reject jobs that don't specify any data.`,
			},
			{
				FlagName:     "job-selection-accept-networked",
				ConfigPath:   NodeRequesterJobSelectionPolicyAcceptNetworkedJobs,
				DefaultValue: config_v2.Default.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
				Description:  `Accept jobs that require network access.`,
			},
			{
				FlagName:     "job-selection-probe-http",
				ConfigPath:   NodeRequesterJobSelectionPolicyProbeHTTP,
				DefaultValue: config_v2.Default.Node.Requester.JobSelectionPolicy.ProbeHTTP,
				Description:  `Use the result of a HTTP POST to decide if we should take on the job.`,
			},
			{
				FlagName:     "job-selection-probe-exec",
				ConfigPath:   NodeRequesterJobSelectionPolicyProbeExec,
				DefaultValue: config_v2.Default.Node.Requester.JobSelectionPolicy.ProbeExec,
				Description:  `Use the result of a exec an external program to decide if we should take on the job.`,
			},
		},
		"labels": []FlagDefinition{
			{
				FlagName:     "labels",
				ConfigPath:   NodeLabels,
				DefaultValue: config_v2.Default.Node.Labels,
				Description:  `Labels to be associated with the node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2)`,
			},
		},
		"libp2p": []FlagDefinition{
			{
				FlagName:     "peer",
				ConfigPath:   NodeLibp2pPeerConnect,
				DefaultValue: config_v2.Default.Node.Libp2p.PeerConnect,
				Description: `A comma-separated list of libp2p multiaddress to connect to. ` +
					`Use "none" to avoid connecting to any peer, ` +
					`"env" to connect to the default peer list of your active environment (see BACALHAU_ENVIRONMENT env var).`,
			},
			{
				FlagName:     "swarm-port",
				ConfigPath:   NodeLibp2pSwarmPort,
				DefaultValue: config_v2.Default.Node.Libp2p.SwarmPort,
				Description:  `The port to listen on for swarm connections.`,
			},
		},
		"list-local-paths": []FlagDefinition{
			{
				FlagName:     "allow-listed-local-paths",
				ConfigPath:   NodeAllowListedLocalPaths,
				DefaultValue: config_v2.Default.Node.AllowListedLocalPaths,
				Description:  "Local paths that are allowed to be mounted into jobs",
			},
		},
		"node-type": []FlagDefinition{
			{
				FlagName:     "node-type",
				ConfigPath:   NodeType,
				DefaultValue: config_v2.Default.Node.Type,
				Description:  `Whether the node is a compute, requester or both.`,
			},
		},
	}

}
