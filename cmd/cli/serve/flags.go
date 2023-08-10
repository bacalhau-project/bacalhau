package serve

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/pkg/config"
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
		Path:        config.NodeDisabledFeaturesEngines,
		Default:     config.Default.Node.DisabledFeatures.Engines,
		Description: "Engine types to disable",
	},
	{
		Name:        config.NodeDisabledFeaturesPublishers,
		Path:        "Node.DisabledFeature.Publishers",
		Default:     config.Default.Node.DisabledFeatures.Publishers,
		Description: "Engine types to disable",
	},
	{
		Name:        "disable-storage",
		Path:        config.NodeDisabledFeaturesStorages,
		Default:     config.Default.Node.DisabledFeatures.Storages,
		Description: "Engine types to disable",
	},
}

var CapacityFlags = []flagDefinition{
	{
		Name:        "job-execution-timeout-bypass-client-id",
		Path:        config.NodeComputeClientIDBypass,
		Default:     config.Default.Node.Compute.ClientIDBypass,
		Description: `List of IDs of clients that are allowed to bypass the job execution timeout check`,
	},
	{
		Name:        "limit-total-cpu",
		Path:        config.NodeComputeCapacityTotalCPU,
		Default:     config.Default.Node.Compute.Capacity.TotalCPU,
		Description: `Total CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	},
	{
		Name:        "limit-total-memory",
		Path:        config.NodeComputeCapacityTotalMemory,
		Default:     config.Default.Node.Compute.Capacity.TotalMemory,
		Description: `Total Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	},
	{
		Name:        "limit-total-gpu",
		Path:        config.NodeComputeCapacityTotalGPU,
		Default:     config.Default.Node.Compute.Capacity.TotalGPU,
		Description: `Total GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	},
	{
		Name:        "limit-job-cpu",
		Path:        config.NodeComputeCapacityJobCPU,
		Default:     config.Default.Node.Compute.Capacity.JobCPU,
		Description: `Job CPU core limit to run all jobs (e.g. 500m, 2, 8).`,
	},
	{
		Name:        "limit-job-memory",
		Path:        config.NodeComputeCapacityJobMemory,
		Default:     config.Default.Node.Compute.Capacity.JobMemory,
		Description: `Job Memory limit to run all jobs  (e.g. 500Mb, 2Gb, 8Gb).`,
	},
	{
		Name:        "limit-job-gpu",
		Path:        config.NodeComputeCapacityJobGPU,
		Default:     config.Default.Node.Compute.Capacity.JobGPU,
		Description: `Job GPU limit to run all jobs (e.g. 1, 2, or 8).`,
	},
}

var EstuaryFlags = []flagDefinition{
	{
		Name:        "estuary-api-key",
		Path:        config.NodeEstuaryAPIKey,
		Default:     config.Default.Node.EstuaryAPIKey,
		Description: `The API key used when using the estuary API.`,
	},
}

var IPFSFlags = []flagDefinition{
	{
		Name:        "ipfs-swarm-addr",
		Path:        config.NodeIPFSSwarmAddresses,
		Default:     config.Default.Node.IPFS.SwarmAddresses,
		Description: "IPFS multiaddress to connect the in-process IPFS node to - cannot be used with --ipfs-connect.",
	},
	{
		Name:        "ipfs-connect",
		Path:        config.NodeIPFSConnect,
		Default:     config.Default.Node.IPFS.Connect,
		Description: "The ipfs host multiaddress to connect to, otherwise an in-process IPFS node will be created if not set.",
	},
	{
		Name:    "private-internal-ipfs",
		Path:    config.NodeIPFSPrivateInternal,
		Default: config.Default.Node.IPFS.PrivateInternal,
		Description: "Whether the in-process IPFS node should auto-discover other nodes, including the public IPFS network - " +
			"cannot be used with --ipfs-connect. " +
			"Use \"--private-internal-ipfs=false\" to disable. " +
			"To persist a local Ipfs node, set BACALHAU_SERVE_IPFS_PATH to a valid path.",
	},
}

var JobSelectionFlags = []flagDefinition{
	{
		Name:        "job-selection-data-locality",
		Path:        config.NodeRequesterJobSelectionPolicyLocality,
		Default:     config.Default.Node.Requester.JobSelectionPolicy.Locality,
		Description: `Only accept jobs that reference data we have locally ("local") or anywhere ("anywhere").`,
	},
	{
		Name:        "job-selection-reject-stateless",
		Path:        config.NodeRequesterJobSelectionPolicyRejectStatelessJobs,
		Default:     config.Default.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
		Description: `Reject jobs that don't specify any data.`,
	},
	{
		Name:        "job-selection-accept-networked",
		Path:        config.NodeRequesterJobSelectionPolicyAcceptNetworkedJobs,
		Default:     config.Default.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
		Description: `Accept jobs that require network access.`,
	},
	{
		Name:        "job-selection-probe-http",
		Path:        config.NodeRequesterJobSelectionPolicyProbeHTTP,
		Default:     config.Default.Node.Requester.JobSelectionPolicy.ProbeHTTP,
		Description: `Use the result of a HTTP POST to decide if we should take on the job.`,
	},
	{
		Name:        "job-selection-probe-exec",
		Path:        config.NodeRequesterJobSelectionPolicyProbeExec,
		Default:     config.Default.Node.Requester.JobSelectionPolicy.ProbeExec,
		Description: `Use the result of a exec an external program to decide if we should take on the job.`,
	},
}

var LabelFlags = []flagDefinition{
	{
		Name:        "labels",
		Path:        config.NodeLabels,
		Default:     config.Default.Node.Labels,
		Description: `Labels to be associated with the node that can be used for node selection and filtering. (e.g. --labels key1=value1,key2=value2)`,
	},
}

var Libp2pFlags = []flagDefinition{
	{
		Name:    "peer",
		Path:    config.NodeLibp2pPeerConnect,
		Default: config.Default.Node.Libp2p.PeerConnect,
		Description: `A comma-separated list of libp2p multiaddress to connect to. ` +
			`Use "none" to avoid connecting to any peer, ` +
			`"env" to connect to the default peer list of your active environment (see BACALHAU_ENVIRONMENT env var).`,
	},
	{
		Name:        "swarm-port",
		Path:        config.NodeLibp2pSwarmPort,
		Default:     config.Default.Node.Libp2p.SwarmPort,
		Description: `The port to listen on for swarm connections.`,
	},
}

var AllowListLocalPathsFlags = []flagDefinition{
	{
		Name:        "allow-listed-local-paths",
		Path:        config.NodeAllowListedLocalPaths,
		Default:     config.Default.Node.AllowListedLocalPaths,
		Description: "Local paths that are allowed to be mounted into jobs",
	},
}

var NodeTypeFlags = []flagDefinition{
	{
		Name:        "node-type",
		Path:        config.NodeType,
		Default:     config.Default.Node.Type,
		Description: `Whether the node is a compute, requester or both.`,
	},
}
