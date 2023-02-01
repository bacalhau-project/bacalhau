//go:build windows

package logger

import "strings"

func trimRepositoryRootDir(root string) string {
	return strings.ReplaceAll(root, "\\", "/")
}
