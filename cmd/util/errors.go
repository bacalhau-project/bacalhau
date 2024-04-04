package util

import (
	"math"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/bad"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mitchellh/go-wordwrap"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var red = color.New(color.FgRed)

const errorPrefix = "Error: "

// Print an error in a pretty format, with a prefix and subsequent error text
// wrapped to the size of the terminal and indented.
func PrintErr(cmd *cobra.Command, err *bad.Error) {
	terminalWdith, _, termErr := term.GetSize(int(os.Stderr.Fd()))
	if termErr != nil || terminalWdith <= 0 {
		log.Ctx(cmd.Context()).Debug().Err(termErr).Msg("Failed to get terminal size")
		terminalWdith = math.MaxInt32
	}

	errorWidth := uint(terminalWdith) - uint(len(errorPrefix))
	for _, leafErr := range err.Leaves() {
		red.Fprint(cmd.ErrOrStderr(), errorPrefix)
		for i, line := range strings.Split(wordwrap.WrapString(leafErr.Type, errorWidth), "\n") {
			if i > 0 {
				cmd.PrintErr(strings.Repeat(" ", len(errorPrefix)))
			}
			cmd.PrintErrln(line)
		}
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

// Print an error in the specified output format, or regular pretty format if
// the user is expecting a human-readable table.
//
// If users have asked for JSON/YAML/CSV, they are probably best able to process
// the erorr in those formats as well. But if they asked for a table (the
// default) they are probably doing human processing, so our regular pretty
// output will be most helpful.
func PrintErrFormatted(cmd *cobra.Command, options output.OutputOptions, err *bad.Error) {
	if options.Format == output.TableFormat {
		PrintErr(cmd, err)
	}

	printErr := output.Output(cmd, errorColumns, options, err.Leaves())
	if printErr != nil {
		cmd.PrintErrln(printErr)
	}
}
