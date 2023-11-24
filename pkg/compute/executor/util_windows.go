//go:build windows
// +build windows

package executor

// isFileExecutable checks if the provided file is executable, but
// on windows we just assume it is.
func isFileExecutable(absFilePath string) (bool, string) {
	return true, ""
}
