package system

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

// EmbeddedFS holds static binary.
//go:generate cp -r ../../embed .
//go:embed embed/shellcheck
var EmbeddedFS embed.FS

// TODO: This should go away - if we did this correctly, we'd build an embed the haskell library into the go binary
// and figure out how to call it with cgo. However, this is cheap and cheerful enough for now.
func GetShellCheckerBinary() (string, error) {

	shellCheckBinaryName := "shellcheck"
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Error().Err(err).Msgf("Could not access user cache directory.")
		return "", err
	}

	shellCheckBinaryFullPath := filepath.Join(cacheDir, shellCheckBinaryName)
	shellCheckBinary, err := os.Stat(shellCheckBinaryFullPath)

	// Check to see if the file exists and is executable.
	if !errors.Is(err, os.ErrNotExist) && (shellCheckBinary.Mode()&0111 != 0) {
		return shellCheckBinaryFullPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Error().Err(err).Msgf("Could not check cache directory for executable binary.")
		return "", err
	}

	binaryByteArray, err := EmbeddedFS.ReadFile(filepath.Join("embed", shellCheckBinaryName))
	if err != nil {
		log.Error().Err(err).Msgf("Could not read shellchecker binary in embedded directory.")
		return "", err
	}

	err = os.WriteFile(shellCheckBinaryFullPath, binaryByteArray, fs.FileMode(0111))

	if err != nil {
		log.Error().Err(err).Msgf("Could not write shellchecker to the cache directory.")
		return "", err
	}

	return shellCheckBinaryFullPath, nil
}

func CheckBashSyntax(cmds []string) (error, *bytes.Buffer, *bytes.Buffer) {
	if runtime.GOOS != "linux" {
		log.Warn().Msg("Unfortunately, shellchecking is only available on Linux. Continuing.")
		return nil, nil, nil
	}

	shellChecker, err := GetShellCheckerBinary()

	if err != nil {
		msg := "Cannot create shellchecker binary"
		log.Error().Err(err).Msg(msg)
		return err, nil, nil
	}

	file, err := ioutil.TempFile("", "bacalhau-script-checker*.sh")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not create temp file for script")
	}
	defer os.Remove(file.Name())

	_, err = file.WriteString(strings.Join(cmds, "\n"))
	if err != nil {
		log.Fatal().Err(err).Msg("Could not write script to temp file")
	}

	return RunTeeCommand(shellChecker, []string{"--norc", "--shell=bash", "--severity=warning", file.Name()})
}
