package logs

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/cli/job"

	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	logsShortDesc = templates.LongDesc(i18n.T(`
		Follow logs from a currently executing job
`))

	//nolint:lll // Documentation
	logsExample = templates.Examples(i18n.T(`
		# Follow logs for a previously submitted job
		bacalhau logs j-51225160-807e-48b8-88c9-28311c7899e1

		# Follow output with a short ID
		bacalhau logs j-ebd9bf2f
`))
)

func NewCmd() *cobra.Command {
	cmd := job.NewLogCmd()
	cmd.Short = logsShortDesc
	cmd.Example = logsExample
	return cmd
}
