package logs

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

	//nolint:lll // Documentation
	logsExample = templates.Examples(i18n.T(`
		# Follow logs for a previously submitted job
		bacalhau logs 51225160-807e-48b8-88c9-28311c7899e1

		# Follow output with a short ID 
		bacalhau logs ebd9bf2f
`))
)

type LogCommandOptions struct {
	Follow      bool
	WithHistory bool
}

func NewCmd() *cobra.Command {
	options := LogCommandOptions{}

	logsCmd := &cobra.Command{
		Use:     "logs [id]",
		Short:   logsShortDesc,
		Example: logsExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  util.ApplyPorcelainLogLevel,
		Run: func(cmd *cobra.Command, cmdArgs []string) {
			if err := util.Logs(cmd, cmdArgs[0], options.Follow, options.WithHistory); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	logsCmd.PersistentFlags().BoolVarP(
		&options.Follow, "follow", "f", false,
		`Follow the logs in real-time after retrieving the current logs.`,
	)

	logsCmd.PersistentFlags().BoolVarP(
		&options.Follow, "history", "h", false,
		`Show all logs with history. TODO better definition`,
	)

	return logsCmd
}
