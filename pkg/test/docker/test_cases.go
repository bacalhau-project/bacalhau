package docker

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

type TestCase struct {
	Name           string
	SetupStorage   ISetupStorage
	ResultsChecker ICheckResults
	GetJobSpec     IGetJobSpec
}

func GetTestCases(
	t *testing.T,
) []TestCase {
	MOUNT_PATH := "/data/file.txt"
	OUTPUT_FILE := "output_file.txt"
	HELLO_WORLD := `hello world`
	testCases := []TestCase{
		{
			Name: "cat_file",
			SetupStorage: singleFileSetupStorageWithData(
				t,
				HELLO_WORLD,
				MOUNT_PATH,
			),
			ResultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				HELLO_WORLD,
				ExpectedModeEquals,
				1,
			),
			GetJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				return types.JobSpecVm{
					Image: "ubuntu:latest",
					Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
						"cat",
						MOUNT_PATH,
					}),
				}
			},
		},
		{
			Name: "grep_file",
			SetupStorage: singleFileSetupStorageWithFile(
				t,
				"../../../testdata/grep_file.txt",
				MOUNT_PATH,
			),
			ResultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				"kiwi is delicious",
				ExpectedModeContains,
				2,
			),
			GetJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				return types.JobSpecVm{
					Image: "ubuntu:latest",
					Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
						"grep",
						"kiwi",
						MOUNT_PATH,
					}),
				}
			},
		},
		{
			Name: "sed_file",
			SetupStorage: singleFileSetupStorageWithFile(
				t,
				"../../../testdata/sed_file.txt",
				MOUNT_PATH,
			),
			ResultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				"LISBON",
				ExpectedModeContains,
				5,
			),
			GetJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				return types.JobSpecVm{
					Image: "ubuntu:latest",
					Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
						"sed",
						"-n",
						"/38.7[2-4]..,-9.1[3-7]../p",
						MOUNT_PATH,
					}),
				}
			},
		},
		{
			Name: "awk_file",
			SetupStorage: singleFileSetupStorageWithFile(
				t,
				"../../../testdata/awk_file.txt",
				MOUNT_PATH,
			),
			ResultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				"LISBON",
				ExpectedModeContains,
				501,
			),
			GetJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				// TODO: work out why we need this extra quote
				// when we bash -c the command (because we are in test output volumes mode)
				extraQuote := ""
				if outputMode == OutputModeVolume {
					extraQuote = "'"
				}
				return types.JobSpecVm{
					Image: "ubuntu:latest",
					Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
						"awk",
						"-F,",
						extraQuote + "{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}" + extraQuote,
						MOUNT_PATH,
					}),
				}
			},
		},
	}

	return testCases
}
