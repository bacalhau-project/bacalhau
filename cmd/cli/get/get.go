package get

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/cli/job"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	getExample = templates.Examples(i18n.T(`
		# Get the results of a job.
		bacalhau get 51225160-807e-48b8-88c9-28311c7899e1

		# Get the results of a job, with a short ID.
		bacalhau get ebd9bf2f
`))
)

type GetOptions struct {
	DownloadSettings *cliflags.DownloaderSettings
}

func NewCmd() *cobra.Command {
	cmd := job.NewGetCmd()
	cmd.Example = getExample
	return cmd
}
