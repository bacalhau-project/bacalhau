package job

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
)

var (
	logsShortDesc = templates.LongDesc(`
		Follow logs from a currently executing job
`)

	logsExample = templates.Examples(`
		# Read logs for a previously submitted job
		bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1

		# Follow logs for a previously submitted job
		bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1 --follow

		# Tail logs for a previously submitted job
		bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1 --tail
`)
)

type LogCommandOptions struct {
	ExecutionID string
	Follow      bool
	Tail        bool
}

func NewLogCmd() *cobra.Command {
	options := LogCommandOptions{}

	logsCmd := &cobra.Command{
		Use:           "logs [id]",
		Short:         logsShortDesc,
		Example:       logsExample,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			opts := util.LogOptions{
				JobID:       cmdArgs[0],
				ExecutionID: options.ExecutionID,
				Follow:      options.Follow,
				Tail:        options.Tail,
			}
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.NewAPIClientManager(cmd, cfg).GetAuthenticatedAPIClient()
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return util.Logs(cmd, api, opts)
		},
	}

	logsCmd.PersistentFlags().StringVarP(
		&options.ExecutionID, "execution-id", "e", "",
		"Retrieve logs from a specific execution of the job.",
	)

	logsCmd.PersistentFlags().BoolVarP(
		&options.Follow, "follow", "f", false,
		`Follow the logs in real-time after retrieving the current logs.`,
	)

	logsCmd.PersistentFlags().BoolVarP(
		&options.Tail, "tail", "t", false,
		"Tail the logs from the end of the log stream.",
	)
	return logsCmd
}
