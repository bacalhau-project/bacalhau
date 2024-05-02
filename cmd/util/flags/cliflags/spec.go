package cliflags

import (
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/util/opts"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

const InputUsageMsg = `Mount URIs as inputs to the job. Can be specified multiple times. Format: src=URI,dst=PATH[,opt=key=value]
Examples:
# Mount IPFS CID to /inputs directory
-i ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72

# Mount S3 object to a specific path
-i s3://bucket/key,dst=/my/input/path

# Mount S3 object with specific endpoint and region
-i src=s3://bucket/key,dst=/my/input/path,opt=endpoint=https://s3.example.com,opt=region=us-east-1
`

func NewSpecFlagDefaultSettings() *SpecFlagSettings {
	return &SpecFlagSettings{
		Publisher:     opts.NewPublisherOpt(),
		Inputs:        opts.StorageOpt{},
		OutputVolumes: map[string]string{},
		EnvVar:        []string{},
		Timeout:       int64(model.DefaultJobTimeout.Seconds()),
		Labels:        []string{},
		Selector:      "",
		DoNotTrack:    false,
	}
}

type SpecFlagSettings struct {
	Publisher     opts.PublisherOpt // Publisher - publisher.Publisher
	Inputs        opts.StorageOpt   // Array of inputs
	OutputVolumes map[string]string // Map of output volumes in 'name:mount point' form
	EnvVar        []string          // Array of environment variables
	Timeout       int64             // Job execution timeout in seconds
	Labels        []string          // Labels for the job on the Bacalhau network (for searching)
	Selector      string            // Selector (label query) to filter nodes on which this job can be executed
	DoNotTrack    bool
}

func SpecFlags(settings *SpecFlagSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Spec settings", pflag.ContinueOnError)
	flags.VarP(
		&settings.Publisher,
		"publisher",
		"p",
		"Where to publish the result of the job",
	)
	flags.VarP(
		&settings.Inputs,
		"input",
		"i",
		InputUsageMsg,
	)
	flags.StringToStringVarP(
		&settings.OutputVolumes,
		"output",
		"o",
		settings.OutputVolumes,
		`name=path of the output data volumes. `+
			`'outputs=/outputs' is always added unless '/outputs' is mapped to a different name.`,
	)
	flags.StringSliceVarP(
		&settings.EnvVar,
		"env",
		"e",
		settings.EnvVar,
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	flags.Int64Var(
		&settings.Timeout,
		"timeout",
		settings.Timeout,
		`Job execution timeout in seconds (e.g. 300 for 5 minutes)`,
	)
	flags.StringSliceVarP(
		&settings.Labels,
		"labels",
		"l",
		settings.Labels,
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)
	flags.StringVarP(
		&settings.Selector,
		"selector",
		"s",
		settings.Selector,
		`Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.`, //nolint:lll // Documentation, ok if long.
	)
	flags.BoolVar(
		&settings.DoNotTrack,
		"do-not-track",
		settings.DoNotTrack,
		// TODO(forrest): we need a better definition of this
		`When true the job will not be tracked(?) TODO BETTER DEFINITION`,
	)

	return flags
}
