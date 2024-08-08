package types

// Volume represents a location on disk with specific attributes.
// It includes a name for reference, the path on the filesystem,
// and a flag indicating if the volume is writable.
type Volume struct {
	// Name is the identifier used to refer to this volume.
	Name string
	// Path is the filesystem location of the volume on disk.
	Path string
	// Write indicates whether the volume has write permissions.
	Write bool
}
