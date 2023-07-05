package flags

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/util/output"
)

func OutputFormatFlags(format *output.OutputOptions) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Output Format", pflag.ContinueOnError)

	flags.Var(OutputFormatFlag(&format.Format), "output",
		fmt.Sprintf(`The output format for the command (one of %q)`, output.AllFormats))
	flags.BoolVar(&format.HideHeader, "hide-header", format.HideHeader,
		`do not print the column headers.`)
	flags.BoolVar(&format.NoStyle, "no-style", format.NoStyle,
		`remove all styling from table output.`)
	flags.BoolVar(&format.Wide, "wide", format.Wide,
		`Print full values in the table results`)

	return flags
}
