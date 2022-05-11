package docker

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
)

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
		// TODO: this test fails because of quoting issues
		// {
		// 	name: "awk_file",
		// 	setupStorage: singleFileSetupStorageWithFile(
		// 		t,
		// 		"../../../testdata/awk_file.txt",
		// 		MOUNT_PATH,
		// 	),
		// 	resultsChecker: singleFileResultsCheckerContains(
		// 		t,
		// 		OUTPUT_FILE,
		// 		"LISBON",
		// 		ExpectedModeContains,
		// 		501,
		// 	),
		// 	getJobSpec: func(outputMode IOutputMode) types.JobSpecVm {
		// 		return types.JobSpecVm{
		// 			Image: "ubuntu",
		// 			Entrypoint: convertEntryPoint(outputMode, OUTPUT_FILE, []string{
		// 				"awk",
		// 				"-F,",
		// 				"'{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}'",
		// 				MOUNT_PATH,
		// 			}),
		// 		}
		// 	},
		// },
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
