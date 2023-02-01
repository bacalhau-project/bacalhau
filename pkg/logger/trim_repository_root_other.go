//go:build !windows

package logger

func trimRepositoryRootDir(root string) string {
	return root
}
