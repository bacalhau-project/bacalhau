package model

//go:generate stringer -type=APIVersion
type APIVersion int

const (
	Unknown APIVersion = iota // must be first
	V1alpha1
)
