package model

//go:generate stringer -type=APIVersion
type JobAPIVersion string

const (
	V1alpha1 JobAPIVersion = "v1alpha1"
)
