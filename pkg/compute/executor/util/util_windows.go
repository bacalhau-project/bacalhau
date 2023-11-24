//go:build windows
// +build windows

package util

// isFileExecutable checks if the provided file is executable, but
// on windows we just assume it is.
func IsFileExecutable(absFilePath string) (bool, string) {
	return true, ""
}
