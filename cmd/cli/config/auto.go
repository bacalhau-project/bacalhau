package config

import (
	"context"
	"fmt"
	"strconv"

	"github.com/BTBurke/k8sresource"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	defaultTotalPercentage      = 75
	defaultJobPercentage        = 75
	defaultDefaultJobPercentage = 75
	defaultQueuePercentage      = defaultJobPercentage * 2

	oneHundredPercent = 100
)

type autoSettings struct {
	TotalPercentage   int
	JobPercentage     int
	DefaultPercentage int
	QueuePercentage   int
}

func autoFlags(settings *autoSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Auto configuration setting", pflag.ContinueOnError)
	flags.IntVar(&settings.TotalPercentage,
		"total-percentage",
		defaultTotalPercentage,
		//nolint:goconst
		"Percentage expressed as a number from 1 to 100 representing "+
			"total amount of resource the system can be using at one time in aggregate for all jobs "+
			"(values over 100 will be rejected)")
	flags.IntVar(&settings.JobPercentage,
		"job-percentage",
		defaultJobPercentage,
		"Percentage expressed as a number from 1 to 100 representing "+
			"per job amount of resource the system can be using at one time for a single job "+
			"(values over 100 will be rejected)")
	flags.IntVar(&settings.DefaultPercentage,
		"default-job-percentage",
		defaultDefaultJobPercentage,
		"Percentage expressed as a number from 1 to 100 representing "+
			"default per job amount of resources jobs will get when they don't specify any resource limits themselves "+
			"(values over 100 will be rejected")
	flags.IntVar(&settings.QueuePercentage,
		"queue-job-percentage",
		defaultQueuePercentage,
		"Percentage expressed as a number from 1 to 100 representing the total amount of resource the system "+
			"can queue at one time in aggregate for all jobs (values over 100 are accepted)")
	return flags
}

func (a *autoSettings) validate() error {
	if a.TotalPercentage == 0 {
		return fmt.Errorf("total-percentage must be greater than 0")
	}
	if a.TotalPercentage > oneHundredPercent {
		return fmt.Errorf("total-percentage cannot exceed 100")
	}
	if a.JobPercentage == 0 {
		return fmt.Errorf("job-percentage must be greater than 0")
	}
	if a.JobPercentage > oneHundredPercent {
		return fmt.Errorf("job-percentage cannot exceed 100")
	}
	if a.JobPercentage > a.TotalPercentage {
		return fmt.Errorf("job-percentage must be less than or equal to total-percentage")
	}
	if a.DefaultPercentage == 0 {
		return fmt.Errorf("default-job-percentage must be greater than 0")
	}
	if a.DefaultPercentage > oneHundredPercent {
		return fmt.Errorf("default-job-percentage cannot exceed 100")
	}
	if a.DefaultPercentage > a.TotalPercentage {
		return fmt.Errorf("default-job-percentage must be less than or equal to total-percentage")
	}
	return nil
}

func newAutoResourceCmd() *cobra.Command {
	settings := new(autoSettings)
	autoCmd := &cobra.Command{
		Use:      "auto-resources [flags]",
		Short:    "Auto set compute resources values in the config.",
		Args:     cobra.MinimumNArgs(0),
		PreRunE:  util.ClientPreRunHooks,
		PostRunE: util.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := settings.validate(); err != nil {
				return err
			}
			return autoConfig(cmd.Context(), settings)
		},
	}

	autoCmd.PersistentFlags().AddFlagSet(autoFlags(settings))

	return autoCmd
}

func autoConfig(ctx context.Context, settings *autoSettings) error {
	pp := system.NewPhysicalCapacityProvider()
	physicalResources, err := pp.GetTotalCapacity(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate system physical resources: %w", err)
	}

	if err := setResources(types.NodeComputeCapacityTotalResourceLimits, settings.TotalPercentage, physicalResources); err != nil {
		return err
	}
	if err := setResources(types.NodeComputeCapacityJobResourceLimits, settings.JobPercentage, physicalResources); err != nil {
		return err
	}
	if err := setResources(types.NodeComputeCapacityDefaultJobResourceLimits, settings.DefaultPercentage, physicalResources); err != nil {
		return err
	}
	if err := setResources(types.NodeComputeCapacityQueueResourceLimits, settings.QueuePercentage, physicalResources); err != nil {
		return err
	}

	return nil
}

func setResources(key string, percentageValue int, resources models.Resources) error {
	percentage := float64(percentageValue) * .01
	if err := setConfig(fmt.Sprintf("%s.CPU", key), k8sresource.NewCPUFromFloat(resources.CPU*percentage).ToString()); err != nil {
		return err
	}
	if err := setConfig(fmt.Sprintf("%s.Memory", key), humanize.Bytes(uint64(float64(resources.Memory)*percentage))); err != nil {
		return err
	}
	if err := setConfig(fmt.Sprintf("%s.Disk", key), humanize.Bytes(uint64(float64(resources.Disk)*percentage))); err != nil {
		return err
	}
	// NB(forrest): we use an int64 to represent number of GPUs on a system, max value is roughly nine quintillion
	// we are downcasting it to an int with a max value of ~two billion, if there are ever this many GPUs let us
	// celebrate the panic this will cause.
	if err := setConfig(fmt.Sprintf("%s.GPU", key), strconv.Itoa(int(resources.GPU))); err != nil {
		return err
	}
	return nil
}
