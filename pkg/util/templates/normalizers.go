package templates

import (
	"k8s.io/kubectl/pkg/util/templates"
)

// LongDesc normalizes a command's long description to follow the conventions.
func LongDesc(s string) string {
	return templates.LongDesc(s)
}

// Examples normalizes a command's examples to follow the conventions.
func Examples(s string) string {
	return templates.Examples(s)
}
