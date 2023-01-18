package server

// client code for accessing bacalhau, literally ripped straight from the CLI

import (
	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
)

func runStableDiffusion(prompt string) (string, error) {
	runOptions := bacalhau.NewDockerRunOptions()
	runOptions.RunTimeSettings.WaitForJobToFinish = true
	// need to set this to get the cid out
	runOptions.RunTimeSettings.PrintNodeDetails = true
	runOptions.GPU = "1"
	return bacalhau.DockerRun(nil, []string{
		"ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",
		"--", "python", "main.py", "--o", "./outputs", "--p",
		prompt,
	}, runOptions)
}
