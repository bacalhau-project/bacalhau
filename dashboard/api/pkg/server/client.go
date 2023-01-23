package server

// client code for accessing bacalhau, literally ripped straight from the CLI

import (
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/rundocker"
	"github.com/spf13/cobra"
)

var configInitRan bool

func runStableDiffusion(prompt string, testing bool) (string, error) {
	if !configInitRan {
		system.InitConfig()
		configInitRan = true
	}
	runOptions := rundocker.NewDockerRunOptions()
	runOptions.RunTimeSettings.WaitForJobToFinish = true
	// need to set this to get the cid out
	runOptions.RunTimeSettings.PrintNodeDetails = true

	// because the rundocker machinery likes to run cmd.Print{,f}
	nullCommand := &cobra.Command{
		Use:   "null",
		Short: "null",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	// just to fill in contexts, etc... hacks hacks hacks
	nullCommand.Execute()

	if !testing {
		// gpus, actual stable diffusion:
		runOptions.GPU = "1"
		return rundocker.DockerRun(nullCommand, []string{
			"ghcr.io/bacalhau-project/examples/stable-diffusion-gpu:0.0.1",
			"python", "main.py", "--o", "./outputs", "--p",
			prompt,
		}, runOptions)
	} else {
		// testing only:
		return rundocker.DockerRun(nullCommand, []string{
			"ubuntu",
			"echo", prompt,
		}, runOptions)
	}
}
