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
	FRUITS := `apple
orange
pineapple
pear
peach
cherry
kiwi is delicious
strawberry
lemon
raspberry
	`

	tests := []struct {
		name           string
		fileContents   string
		mountPath      string
		expectedOutput string
		expectedMode   IExpectedMode
		getJobSpec     IGetJobSpec
	}{
		{
			name:           "cat_file",
			fileContents:   HELLO_WORLD,
			mountPath:      MOUNT_PATH,
			expectedOutput: HELLO_WORLD,
			expectedMode:   ExpectedModeEquals,
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
			name:           "grep_file",
			fileContents:   FRUITS,
			mountPath:      MOUNT_PATH,
			expectedOutput: "kiwi is delicious",
			expectedMode:   ExpectedModeContains,
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
	}

	for _, test := range tests {

		dockerExecutorStorageTest(
			t,
			test.name,
			singleFileSetupStorage(
				t,
				test.fileContents,
				test.mountPath,
			),
			singleFileResultsChecker(
				t,
				test.expectedOutput,
				test.expectedMode,
				OUTPUT_FILE,
			),
			test.getJobSpec,
		)

	}
}
