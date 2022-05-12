package docker

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

// each of these tests will use both fuse and api copy storage drivers
// as well as stdout vs output volume mode
// so each test will be run 4 times
func TestSingleFile(t *testing.T) {

	MOUNT_PATH := "/data/file.txt"
	OUTPUT_FILE := "output_file.txt"
	HELLO_WORLD := `hello world`

	tests := []struct {
		name           string
		setupStorage   ISetupStorage
		resultsChecker ICheckResults
		getJobSpec     IGetJobSpec
	}{
		{
			name: "cat_file",
			setupStorage: singleFileSetupStorageWithData(
				t,
				HELLO_WORLD,
				MOUNT_PATH,
			),
			resultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				HELLO_WORLD,
				ExpectedModeEquals,
				1,
			),
			getJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				return types.JobSpecVm{
					Image: "ubuntu",
					Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
						"cat",
						MOUNT_PATH,
					}),
				}
			},
		},
		{
			name: "grep_file",
			setupStorage: singleFileSetupStorageWithFile(
				t,
				"../../../testdata/grep_file.txt",
				MOUNT_PATH,
			),
			resultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				"kiwi is delicious",
				ExpectedModeContains,
				2,
			),
			getJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				return types.JobSpecVm{
					Image: "ubuntu",
					Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
						"grep",
						"kiwi",
						MOUNT_PATH,
					}),
				}
			},
		},
		{
			name: "sed_file",
			setupStorage: singleFileSetupStorageWithFile(
				t,
				"../../../testdata/sed_file.txt",
				MOUNT_PATH,
			),
			resultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				"LISBON",
				ExpectedModeContains,
				5,
			),
			getJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				return types.JobSpecVm{
					Image: "ubuntu",
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
			name: "awk_file",
			setupStorage: singleFileSetupStorageWithFile(
				t,
				"../../../testdata/awk_file.txt",
				MOUNT_PATH,
			),
			resultsChecker: singleFileResultsCheckerContains(
				t,
				OUTPUT_FILE,
				"LISBON",
				ExpectedModeContains,
				501,
			),
			getJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
				// TODO: work out why we need this extra quote
				// when we bash -c the command (because we are in test output volumes mode)
				extraQuote := ""
				if outputMode == OutputModeVolume {
					extraQuote = "'"
				}
				return types.JobSpecVm{
					Image: "ubuntu",
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

	for _, test := range tests {

		dockerExecutorStorageTest(
			t,
			test.name,
			test.setupStorage,
			test.resultsChecker,
			test.getJobSpec,
		)

	}
}
