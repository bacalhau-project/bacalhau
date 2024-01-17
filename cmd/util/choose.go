package util

import (
	"bufio"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
)

// Choose offers the user a choice in their terminal of some available options,
// and returns the chosen option when they have made a valid selection. If there
// is only one available option, they won't be offered a choice and it will just
// be returned. It is an error to pass zero options.
func Choose[Choice any](cmd *cobra.Command, prompt string, choices []Choice) (choice Choice, err error) {
	switch len(choices) {
	case 0:
		return choice, fmt.Errorf("no possible choices")
	case 1:
		return choices[0], nil
	default:
		cmd.PrintErrln(prompt)
		for index, name := range choices {
			cmd.PrintErrf("%d. %v\n", index+1, name)
		}

		reader := bufio.NewScanner(cmd.InOrStdin())
		for {
			cmd.PrintErr("Choose a number: ")

			more := reader.Scan()
			if !more && reader.Err() != nil {
				return choice, reader.Err()
			} else if !more {
				return choice, io.EOF
			}

			index, err := strconv.ParseUint(reader.Text(), 10, 32)
			if err != nil {
				cmd.PrintErrf("invalid choice: %s\n", err.Error())
			} else if index < 1 || index > uint64(len(choices)) {
				cmd.PrintErrf("invalid choice: %d\n", index)
			} else {
				return choices[index-1], nil
			}
		}
	}
}
