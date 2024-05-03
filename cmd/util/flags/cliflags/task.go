package cliflags

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/opts"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	NameUsageMsg = `The name to refer to this task by`

	InputUsageMsg = `Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
Examples:
# Mount IPFS CID to /inputs directory
-i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

# Mount S3 object to a specific path
-i s3://bucket/key,dst=/my/input/path

# Mount S3 object with specific endpoint and region
-i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1
`

	ResultPathUsageMsg = "name=path of the output data volumes. 'outputs=/outputs' is always added unless " +
		"'/outputs' is mapped to a different name."

	PublisherUsageMsg = `Where to publish the result of the job`

	ResourceCPUUsageMsg    = `Job CPU cores (e.g. 500m, 2, 8).`
	ResourceMemoryUsageMsg = `Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).`
	ResourceDiskUsageMsg   = `Job Disk requirement (e.g. 500Gb, 2Tb, 8Tb).`
	ResourceGPUUsageMsg    = `Job GPU requirement (e.g. 1, 2, 8).`

	NetworkTypeUsageMsg   = `Networking capability required by the job. None, HTTP, or Full`
	NetworkDomainUsageMsg = `Domain(s) that the job needs to access (for HTTP networking)`
)

type TaskSettings struct {
	Name string
	// TODO Engine, maybe a generic?
	InputSources         opts.StorageOpt
	ResultPaths          map[string]string
	EnvironmentVariables map[string]string
	Publisher            opts.PublisherOpt
	Resources            ResourceSettings
	Network              NetworkSettings
	Timeout              int64
}

type ResourceSettings struct {
	CPU    string
	Memory string
	Disk   string
	GPU    string
}

type NetworkSettings struct {
	Network models.Network
	Domains []string
}

func DefaultTaskSettings() *TaskSettings {
	return &TaskSettings{
		Name:                 idgen.NewTaskName(),
		InputSources:         opts.StorageOpt{},
		ResultPaths:          make(map[string]string),
		EnvironmentVariables: make(map[string]string),
		Publisher:            opts.PublisherOpt{},
		Resources: ResourceSettings{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
		Network: NetworkSettings{
			Network: models.NetworkNone,
			Domains: make([]string, 0),
		},
		Timeout: int64(time.Duration(0)),
	}
}

func RegisterTaskFlags(cmd *cobra.Command, s *TaskSettings) {
	fs := pflag.NewFlagSet("task", pflag.ContinueOnError)
	fs.StringVar(&s.Name, "task-name", s.Name, NameUsageMsg)
	fs.VarP(&s.InputSources, "input", "i", InputUsageMsg)
	fs.StringToStringVarP(&s.ResultPaths, "output", "o", s.ResultPaths, ResultPathUsageMsg)
	fs.VarP(&s.Publisher, "publisher", "p", PublisherUsageMsg)
	fs.StringVar(&s.Resources.CPU, "cpu", s.Resources.CPU, ResourceCPUUsageMsg)
	fs.StringVar(&s.Resources.Memory, "memory", s.Resources.Memory, ResourceMemoryUsageMsg)
	fs.StringVar(&s.Resources.Disk, "disk", s.Resources.Disk, ResourceDiskUsageMsg)
	fs.StringVar(&s.Resources.GPU, "gpu", s.Resources.GPU, ResourceGPUUsageMsg)
	fs.Var(flags.NetworkFlag(&s.Network.Network), "network", NetworkTypeUsageMsg)
	fs.StringArrayVar(&s.Network.Domains, "domain", s.Network.Domains, NetworkDomainUsageMsg)

	cmd.Flags().AddFlagSet(fs)
}
