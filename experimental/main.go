package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/filecoin-project/bacalhau/internal/logger"
)

// Writer type used to initialize buffer writer
type Writer int

func (*Writer) Write(p []byte) (s string) {
	return s
}

func main() {

	tmpFile, _ := ioutil.TempFile("/tmp", "logfile")

	defer tmpFile.Close() //nolint
	defer os.Remove(tmpFile.Name())

	cmd := exec.Command("echo", "foobaz", "bartap")

	// get the stdout and stderr stream
	erc, err := cmd.StderrPipe()
	if err != nil {
		logger.Errorf("Failed to get stderr reader: ", err)
	}
	orc, err := cmd.StdoutPipe()
	if err != nil {
		logger.Errorf("Failed to get stdout reader: ", err)
	}

	// combine stdout and stderror ReadCloser
	rc := io.MultiReader(erc, orc)

	// Prepare the writer
	f, err := os.OpenFile(tmpFile.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		logger.Fatalf("Failed to create file")
	}

	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout

	logger.Debugf("Executing command: %s", cmd.String())

	// Command.Start starts a new go routine
	if err := cmd.Start(); err != nil {
		logger.Fatalf("Failed to start the command: %s", err)
	}

	var bufferRead bytes.Buffer
	teereader := io.TeeReader(rc, &bufferRead)

	// Everything read from r will be copied to stdout.
	// a, _ := io.ReadAll(teereader)

	// b := string(a)

	logger.Debugf("Temp file name: %s", f.Name())

	if _, err := io.Copy(f, teereader); err != nil {
		logger.Fatalf("Failed to stream to file: %s", err)
	}

	if err := cmd.Wait(); err != nil {
		logger.Fatalf("Failed to wait the command to execute: %s", err)
	}

	logger.Debugf("Buffer: %s", bufferRead.String())

	// TODO: Should we check the result here?
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill() // nolint
	}

	f.Name()

}
