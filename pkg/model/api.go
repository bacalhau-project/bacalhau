package model

//go:generate stringer -type=APIVersion
type APIVersion string

const (
	V1alpha1 APIVersion = "v1alpha1" // must be first
)
