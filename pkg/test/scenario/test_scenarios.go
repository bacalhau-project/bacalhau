package scenario

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

const HELLO_WORLD = "hello world"
const SIMPLE_MOUNT_PATH = "/data/file.txt"
const SIMPLE_OUTPUT_PATH = "/output_data/output_file.txt"
const STDOUT = "stdout"
const CAT_PROGRAM = "cat " + SIMPLE_MOUNT_PATH + " > " + SIMPLE_OUTPUT_PATH

func CatFileToStdout(t *testing.T) TestCase {
	return TestCase{
		Name: "cat_file_to_stdout",
		SetupStorage: singleFileSetupStorageWithData(
			t,
			HELLO_WORLD,
			SIMPLE_MOUNT_PATH,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			STDOUT,
			HELLO_WORLD,
			ExpectedModeEquals,
			1,
		),
		GetJobSpec: func() executor.JobSpecVm {
			return executor.JobSpecVm{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"cat",
					SIMPLE_MOUNT_PATH,
				},
			}
		},
	}
}

func CatFileToVolume(t *testing.T) TestCase {
	return TestCase{
		Name: "cat_file_to_volume",
		SetupStorage: singleFileSetupStorageWithData(
			t,
			CAT_PROGRAM,
			SIMPLE_MOUNT_PATH,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			"test/output_file.txt",
			CAT_PROGRAM,
			ExpectedModeEquals,
			1,
		),
		Outputs: []storage.StorageSpec{
			{
				Name: "test",
				Path: "/output_data",
			},
		},
		GetJobSpec: func() executor.JobSpecVm {
			return executor.JobSpecVm{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"bash",
					SIMPLE_MOUNT_PATH,
				},
			}
		},
	}
}

func GrepFile(t *testing.T) TestCase {
	return TestCase{
		Name: "grep_file",
		SetupStorage: singleFileSetupStorageWithFile(
			t,
			"../../../testdata/grep_file.txt",
			SIMPLE_MOUNT_PATH,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			STDOUT,
			"kiwi is delicious",
			ExpectedModeContains,
			2,
		),
		GetJobSpec: func() executor.JobSpecVm {
			return executor.JobSpecVm{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"grep",
					"kiwi",
					SIMPLE_MOUNT_PATH,
				},
			}
		},
	}
}

func SedFile(t *testing.T) TestCase {
	return TestCase{
		Name: "sed_file",
		SetupStorage: singleFileSetupStorageWithFile(
			t,
			"../../../testdata/sed_file.txt",
			SIMPLE_MOUNT_PATH,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			STDOUT,
			"LISBON",
			ExpectedModeContains,
			5,
		),
		GetJobSpec: func() executor.JobSpecVm {
			return executor.JobSpecVm{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"sed",
					"-n",
					"/38.7[2-4]..,-9.1[3-7]../p",
					SIMPLE_MOUNT_PATH,
				},
			}
		},
	}
}

func AwkFile(t *testing.T) TestCase {
	return TestCase{
		Name: "awk_file",
		SetupStorage: singleFileSetupStorageWithFile(
			t,
			"../../../testdata/awk_file.txt",
			SIMPLE_MOUNT_PATH,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			STDOUT,
			"LISBON",
			ExpectedModeContains,
			501,
		),
		GetJobSpec: func() executor.JobSpecVm {
			return executor.JobSpecVm{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"awk",
					"-F,",
					"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
					SIMPLE_MOUNT_PATH,
				},
			}
		},
	}
}

func GetAllScenarios(t *testing.T) []TestCase {
	return []TestCase{
		CatFileToStdout(t),
		CatFileToVolume(t),
		GrepFile(t),
		SedFile(t),
		AwkFile(t),
	}
}
