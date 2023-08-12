package serve

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type flagDefinition struct {
	Name        string
	Path        string
	Default     interface{}
	Description string
}

// Generic function to define a flag, set a default value, and bind it to Viper
func registerFlags(cmd *cobra.Command, register map[string][]flagDefinition) error {
	for name, defs := range register {
		fset := pflag.NewFlagSet(name, pflag.ContinueOnError)
		// Determine the type of the default value
		for _, def := range defs {
			switch v := def.Default.(type) {
			case int:
				fset.Int(def.Name, v, def.Description)
			case bool:
				fset.Bool(def.Name, v, def.Description)
			case string:
				fset.String(def.Name, v, def.Description)
			case []string:
				fset.StringSlice(def.Name, v, def.Description)
			case map[string]string:
				fset.StringToString(def.Name, v, def.Description)
			case model.JobSelectionDataLocality:
				fset.Var(flags.DataLocalityFlag(&v), def.Name, def.Description)
			default:
				return fmt.Errorf("unhandled type: %T", v)
			}
			if err := viper.BindPFlag(def.Path, fset.Lookup(def.Name)); err != nil {
				return err
			}
		}
		cmd.PersistentFlags().AddFlagSet(fset)
	}
	return nil
}

var DisabledFeatureFlags = []flagDefinition{
	{
		Name:        "disable-engine",
		Path:        types.NodeDisabledFeaturesEngines,
		Default:     types.Default.Node.DisabledFeatures.Engines,
		Description: "Engine types to disable",
	},
	{
		Name:        types.NodeDisabledFeaturesPublishers,
		Path:        "Node.DisabledFeature.Publishers",
		Default:     types.Default.Node.DisabledFeatures.Publishers,
		Description: "Engine types to disable",
	},
	{
		Name:        "disable-storage",
		Path:        types.NodeDisabledFeaturesStorages,
		Default:     types.Default.Node.DisabledFeatures.Storages,
		Description: "Engine types to disable",
	},
}

var CapacityFlags = []flagDefinition{
	{
		Name:        "job-execution-timeout-bypass-client-id",
		Path:        types.NodeComputeJobTimeoutsJobExecutionTimeoutClientIDBypassList,
		Default:     types.Default.Node.Compute.JobTimeouts.JobExecutionTimeoutClientIDBypassList,
		Description: `List of IDs of clients that are allowed to bypass the job execution timeout check`,
	},
	{
		Name:        "limit-total-cpu",
		Path:        types.NodeComputeCapacityTotalResourceLimitsCPU,
		Default:     types.Default.Node.Compute.Capacity.TotalResourceLimits.CPU,
		Description: `Total CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	},
	{
		Name:        "limit-total-memory",
		Path:        types.NodeComputeCapacityTotalResourceLimitsMemory,
		Default:     types.Default.Node.Compute.Capacity.TotalResourceLimits.Memory,
		Description: `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	},
	{
		Name:        "limit-total-gpu",
		Path:        types.NodeComputeCapacityTotalResourceLimitsGPU,
		Default:     types.Default.Node.Compute.Capacity.TotalResourceLimits.GPU,
		Description: `Total GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	},
	{
		Name:        "limit-job-cpu",
		Path:        types.NodeComputeCapacityJobResourceLimitsCPU,
		Default:     types.Default.Node.Compute.Capacity.JobResourceLimits.CPU,
		Description: `Job CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	},
	{
		Name:        "limit-job-memory",
		Path:        types.NodeComputeCapacityDefaultJobResourceLimitsMemory,
		Default:     types.Default.Node.Compute.Capacity.JobResourceLimits.Memory,
		Description: `Job Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	},
	{
		Name:        "limit-job-gpu",
		Path:        types.NodeComputeCapacityJobResourceLimitsGPU,
		Default:     types.Default.Node.Compute.Capacity.JobResourceLimits.GPU,
		Description: `Job GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	},
}

var EstuaryFlags = []flagDefinition{
	{
		Name:        "estuary-api-key",
		Path:        types.NodeEstuaryAPIKey,
		Default:     types.Default.Node.EstuaryAPIKey,
		Description: `The API key used when using the estuary API.`,
	},
}

var IPFSFlags = []flagDefinition{
	{
		Name:        "ipfs-swarm-addr",
		Path:        types.NodeIPFSSwarmAddresses,
		Default:     types.Default.Node.IPFS.SwarmAddresses,
		Description: "IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.",
	},
	{
		Name:        "ipfs-connect",
		Path:        types.NodeIPFSConnect,
		Default:     types.Default.Node.IPFS.Connect,
		Description: "The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.",
	},
	{
		Name:    "private-internal-ipfs",
		Path:    types.NodeIPFSPrivateInternal,
		Default: types.Default.Node.IPFS.PrivateInternal,
		Description: "Whether the in-process IPFS node should auto-discover other nodes, including the public IPFS network - " +
			"cannot be used with --ipfs-connect. " +
			"Use \"--private-internal-ipfs=false\" to disable. " +
			"To persist a local Ipfs node, set BACALHAU_SERVE_IPFS_PATH to a valid path.",
	},
}

var JobSelectionFlags = []flagDefinition{
	{
		Name:        "job-selection-data-locality",
		Path:        types.NodeRequesterJobSelectionPolicyLocality,
		Default:     types.Default.Node.Requester.JobSelectionPolicy.Locality,
		Description: `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	},
	{
		Name:        "job-selection-reject-stateless",
		Path:        types.NodeRequesterJobSelectionPolicyRejectStatelessJobs,
		Default:     types.Default.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
		Description: `Reject jobs that don't specify any data.`,
	},
	{
		Name:        "job-selection-accept-networked",
		Path:        types.NodeRequesterJobSelectionPolicyAcceptNetworkedJobs,
		Default:     types.Default.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
		Description: `Accept jobs that require network access.`,
	},
	{
		Name:        "job-selection-probe-http",
		Path:        types.NodeRequesterJobSelectionPolicyProbeHTTP,
		Default:     types.Default.Node.Requester.JobSelectionPolicy.ProbeHTTP,
		Description: `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		Name:        "job-selection-probe-exec",
		Path:        types.NodeRequesterJobSelectionPolicyProbeExec,
		Default:     types.Default.Node.Requester.JobSelectionPolicy.ProbeExec,
		Description: `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}

var LabelFlags = []flagDefinition{
	{
		Name:        "labels",
		Path:        types.NodeLabels,
		Default:     types.Default.Node.Labels,
		Description: `Labels to be associated with the node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2)`,
	},
}

var Libp2pFlags = []flagDefinition{
	{
		Name:    "peer",
		Path:    types.NodeLibp2pPeerConnect,
		Default: types.Default.Node.Libp2p.PeerConnect,
		Description: `A comma-separated list of libp2p multiaddress to connect to. ` +
			`Use "none" to avoid connecting to any peer, ` +
			`"env" to connect to the default peer list of your active environment (see BACALHAU_ENVIRONMENT env var).`,
	},
	{
		Name:        "swarm-port",
		Path:        types.NodeLibp2pSwarmPort,
		Default:     types.Default.Node.Libp2p.SwarmPort,
		Description: `The port to listen on for swarm connections.`,
	},
}

var AllowListLocalPathsFlags = []flagDefinition{
	{
		Name:        "allow-listed-local-paths",
		Path:        types.NodeAllowListedLocalPaths,
		Default:     types.Default.Node.AllowListedLocalPaths,
		Description: "Local paths that are allowed to be mounted into jobs",
	},
}

var NodeTypeFlags = []flagDefinition{
	{
		Name:        "node-type",
		Path:        types.NodeType,
		Default:     types.Default.Node.Type,
		Description: `Whether the node is a compute, requester or both.`,
	},
}
