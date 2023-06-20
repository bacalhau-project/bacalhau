package flags

import "github.com/spf13/pflag"

type ResourceUsageSettings struct {
	CPU    string
	Memory string
	Disk   string
	GPU    string
}

func ResourceUsageFlags(settings *ResourceUsageSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Resource settings", pflag.ContinueOnError)
	flags.StringVar(
		&settings.CPU,
		"cpu",
		settings.CPU,
		`Job CPU cores (e.g. 500m, 2, 8).`,
	)
	flags.StringVar(
		&settings.Memory,
		"memory",
		settings.Memory,
		`Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	flags.StringVar(
		&settings.Disk,
		"disk",
		settings.Disk,
		`Job Disk requirement (e.g. 500Gb, 2Tb, 8Tb).`,
	)
	flags.StringVar(
		&settings.GPU,
		"gpu",
		settings.GPU,
		`Job GPU requirement (e.g. 1, 2, 8).`,
	)
	return flags
}
