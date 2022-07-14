package system

import (
	"strings"

	"github.com/rs/zerolog/log"
	"mvdan.cc/sh/v3/syntax"
)

func CheckBashSyntax(cmds []string) error {
	script := strings.NewReader(strings.Join(cmds, "\n"))
	_, err := syntax.NewParser().Parse(script, "")

	return err
}

// Function for parsing the entrypoint of a docker command.
// Could be more useful in the future (just does globs without shell parsing for now)
func SanitizeImageAndEntrypoint(jobEntrypoint []string) (returnMessages []string, errorIsFatal bool) {
	errorIsFatal = false // Should and everywhere and set to fatal if yes
	shells := strings.Split(`/bin/sh
/bin/bash
/usr/bin/bash
/bin/rbash
/usr/bin/rbash
/usr/bin/sh
/bin/dash
/usr/bin/dash
/usr/bin/tmux
/usr/bin/screen
/bin/zsh
/usr/bin/zsh`, "/n")

	containsGlob := false
	for _, entrypointArg := range jobEntrypoint {
		if strings.ContainsAny(entrypointArg, "*") {
			containsGlob = true
		}
	}

	if containsGlob {
		for _, shell := range shells {
			if strings.Index(strings.TrimSpace(jobEntrypoint[0]), shell) == 0 {
				containsGlob = false
				break
			}
		}
		if containsGlob {
			msg := "We could not help but notice your command contains a glob, but does not start with a shell. This is almost certainly not going to work. To use globs, you must start your command with a shell (e.g. /bin/bash <your command>)." // nolint:lll // error message, ok to be long
			returnMessages = append(returnMessages, msg)
			log.Warn().Msgf(msg)
		}
	}

	return returnMessages, errorIsFatal
}
