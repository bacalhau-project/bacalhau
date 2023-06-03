package localdirectory

import "strings"

type AllowedPath struct {
	Path      string
	ReadWrite bool
}

// string representation of the object.
func (obj AllowedPath) String() string {
	suffix := "ro"
	if obj.ReadWrite {
		suffix = "rw"
	}
	return obj.Path + ":" + suffix
}

func ParseAllowPath(path string) AllowedPath {
	if strings.HasSuffix(path, ":rw") {
		return AllowedPath{
			Path:      strings.TrimSuffix(path, ":rw"),
			ReadWrite: true,
		}
	} else if strings.HasSuffix(path, ":ro") {
		return AllowedPath{
			Path:      strings.TrimSuffix(path, ":ro"),
			ReadWrite: false,
		}
	} else {
		return AllowedPath{
			Path:      path,
			ReadWrite: false,
		}
	}
}

func ParseAllowPaths(paths []string) []AllowedPath {
	allowedPaths := make([]AllowedPath, len(paths))
	for i, path := range paths {
		allowedPaths[i] = ParseAllowPath(path)
	}
	return allowedPaths
}
