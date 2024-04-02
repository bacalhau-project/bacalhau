package job

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	logsShortDesc = templates.LongDesc(i18n.T(`
		Follow logs from a currently executing job
`))

	logsExample = templates.Examples(i18n.T(`
		# Read logs for a previously submitted job
		bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1

		# Follow logs for a previously submitted job
		bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1 --follow

		# Tail logs for a previously submitted job
		bacalhau job logs j-51225160-807e-48b8-88c9-28311c7899e1 --tail
`))
)

type LogCommandOptions struct {
	ExecutionID string
	Follow      bool
	Tail        bool
}

func NewLogCmd() *cobra.Command {
	options := LogCommandOptions{}

	logsCmd := &cobra.Command{
		Use:     "logs [id]",
		Short:   logsShortDesc,
		Example: logsExample,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, cmdArgs []string) {
			opts := util.LogOptions{
				JobID:       cmdArgs[0],
				ExecutionID: options.ExecutionID,
				Follow:      options.Follow,
				Tail:        options.Tail,
			}
			if err := util.Logs(cmd, opts); err != nil {
				util.Fatal(cmd, err, 1)
			}
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
