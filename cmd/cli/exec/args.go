package exec

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// ExtractUnknownArgs extracts any long-form flags (--something) that are not
// currently configured for this command, they must be flags intended for the
// custom job type.
func ExtractUnknownArgs(flags *pflag.FlagSet, args []string) []string {
	unknownArgs := []string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		var field *pflag.Flag

		if arg[0] == '-' {
			if arg[1] == '-' {
				field = flags.Lookup(strings.SplitN(arg[2:], "=", 2)[0])
			} else {
				for _, s := range arg[1:] {
					field = flags.ShorthandLookup(string(s))
					if field == nil {
						break
					}
				}
			}
		} else {
			continue
		}

		if field != nil {
			if field.NoOptDefVal == "" && i+1 < len(args) && field.Value.String() == args[i+1] {
				i++
			}
			continue
		}

		// Make sure we allow `--code=.` and `--code .`
		if !strings.Contains(arg, "=") {
			if i+1 < len(args) {
				if args[i+1][0] != '-' {
					arg = fmt.Sprintf("%s=%s", arg, args[i+1])
				}
			}
		}

		if arg == "--" {
			continue
		}

		unknownArgs = append(unknownArgs, arg)
	}

	return unknownArgs
}

func flagsToMap(flags []string) map[string]string {
	m := make(map[string]string)

	for _, flag := range flags {
		if flag == "--" {
			continue // skip the user escaping the cmd args
		}

		flagString := strings.TrimPrefix(flag, "-")
		flagString = strings.TrimPrefix(flagString, "-") // just in case there's a second -
		parts := strings.SplitN(flagString, "=", 2)
		if len(parts) == 2 {
			// if the flag has no value, it's probably a standalone bool
			m[parts[0]] = parts[1]
		}
	}

	return m
}
