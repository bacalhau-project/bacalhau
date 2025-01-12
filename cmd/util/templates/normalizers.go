// Package templates provides utilities for formatting CLI help text.
// This implementation is inspired by and simplified from kubectl's templates package
// (k8s.io/kubectl/pkg/util/templates) which is licensed under Apache License 2.0.
package templates

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
)

const Indentation = "  "

// LongDesc formats a command's long description
func LongDesc(s string) string {
	if len(s) == 0 {
		return s
	}
	return normalizer{s}.
		heredoc(). // Handle multiline strings nicely
		trim().    // Remove extra whitespace
		string
}

// Examples formats command examples with proper indentation
func Examples(s string) string {
	if len(s) == 0 {
		return s
	}
	return normalizer{s}.
		trim().
		indent().
		string
}

type normalizer struct {
	string
}

func (s normalizer) heredoc() normalizer {
	s.string = heredoc.Doc(s.string)
	return s
}

func (s normalizer) trim() normalizer {
	s.string = strings.TrimSpace(s.string)
	return s
}

func (s normalizer) indent() normalizer {
	var indentedLines []string
	for _, line := range strings.Split(s.string, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			indented := Indentation + trimmed
			indentedLines = append(indentedLines, indented)
		} else {
			indentedLines = append(indentedLines, "")
		}
	}
	s.string = strings.Join(indentedLines, "\n")
	return s
}
