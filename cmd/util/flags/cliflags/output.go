package cliflags

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

func OutputFormatFlags(format *output.OutputOptions) *pflag.FlagSet {
	flagset := pflag.NewFlagSet("Output Format", pflag.ContinueOnError)

	flagset.Var(flags.OutputFormatFlag(&format.Format), "output",
		fmt.Sprintf(`The output format for the command (one of %q)`, output.AllFormats))
	flagset.BoolVar(&format.HideHeader, "hide-header", format.HideHeader,
		`do not print the column headers.`)
	flagset.BoolVar(&format.NoStyle, "no-style", format.NoStyle,
		`remove all styling from table output.`)
	flagset.BoolVar(&format.Wide, "wide", format.Wide,
		`Print full values in the table results`)

	return flagset
}
