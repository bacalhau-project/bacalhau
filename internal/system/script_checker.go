package system

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)


func CheckBashSyntax(cmds []string) (error) {
	script := strings.NewReader(strings.Join(cmds, "\n"))
	_, err := syntax.NewParser().Parse(script, "")

	return err
}
