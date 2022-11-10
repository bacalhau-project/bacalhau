package model

//go:generate stringer -type=APIVersion
type APIVersion int

const (
	unknown APIVersion = iota // must be first
	V1alpha1
	V1beta1
	done // must be last
)

func APIVersionLatest() APIVersion {
	return done - 1
}
