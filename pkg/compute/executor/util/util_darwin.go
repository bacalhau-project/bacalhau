//go:build darwin
// +build darwin

package util

import (
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

const execBits = 0111

func IsFileExecutable(absFilePath string) (bool, string) {
	fileInfo, err := os.Stat(absFilePath)
	if err != nil {
		return false, err.Error()
	}

	m := fileInfo.Mode()

	if !((m.IsRegular()) || (uint32(m&fs.ModeSymlink) == 0)) {
		return false, fmt.Sprintf("'%s' is not a normal file", absFilePath)
	}

	if uint32(m&execBits) == 0 {
		return false, fmt.Sprintf("'%s' is not executable", absFilePath)
	}

	if err = unix.Access(absFilePath, unix.X_OK); err != nil {
		return false, fmt.Sprintf("'%s' is not executable for current user", absFilePath)
	}

	return true, ""
}
