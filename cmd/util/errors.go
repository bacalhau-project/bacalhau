package util

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/bad"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var red = color.New(color.FgRed)

func PrintErr(cmd *cobra.Command, err *bad.Error) {
	for _, leafErr := range err.Leaves() {
		red.Fprint(cmd.ErrOrStderr(), "Error: ")
		fmt.Fprintln(cmd.ErrOrStderr(), leafErr.Type)
	}
}

var errorColumns = []output.TableColumn[bad.Error]{
	{
		ColumnConfig: table.ColumnConfig{Name: "Error"},
		Value:        func(e bad.Error) string { return e.Type },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Subject"},
		Value:        func(e bad.Error) string { return string(e.Subject) },
	},
}

func PrintErrFormatted(cmd *cobra.Command, options output.OutputOptions, err *bad.Error) {
	if options.Format == output.TableFormat {
		PrintErr(cmd, err)
	}

	printErr := output.Output(cmd, errorColumns, options, err.Leaves())
	if printErr != nil {
		cmd.PrintErr(printErr)
	}
}
