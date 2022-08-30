package scenario


import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

const HelloWorld = "hello world"
const SimpleMountPath = "/data/file.txt"
const SimpleOutputPath = "/output_data/output_file.txt"
const stdoutString = "stdout"
const CatProgram = "cat " + SimpleMountPath + " > " + SimpleOutputPath

func CatFileToStdout(t *testing.T) TestCase {
	return TestCase{
		Name: "cat_file_to_stdout",
		SetupStorage: singleFileSetupStorageWithData(
			t,
			HelloWorld,
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			stdoutString,
			HelloWorld,
			ExpectedModeEquals,
			1,
		),
		GetJobSpec: func() executor.JobSpecDocker {
			return executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"cat",
					SimpleMountPath,
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
			CatProgram,
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			"test/output_file.txt",
			CatProgram,
			ExpectedModeEquals,
			1,
		),
		Outputs: []storage.StorageSpec{
			{
				Name: "test",
				Path: "/output_data",
			},
		},
		GetJobSpec: func() executor.JobSpecDocker {
			return executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"bash",
					SimpleMountPath,
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
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			stdoutString,
			"kiwi is delicious",
			ExpectedModeContains,
			2,
		),
		GetJobSpec: func() executor.JobSpecDocker {
			return executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"grep",
					"kiwi",
					SimpleMountPath,
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
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			stdoutString,
			"LISBON",
			ExpectedModeContains,
			5, //nolint:gomnd // magic number ok for testing
		),
		GetJobSpec: func() executor.JobSpecDocker {
			return executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"sed",
					"-n",
					"/38.7[2-4]..,-9.1[3-7]../p",
					SimpleMountPath,
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
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			stdoutString,
			"LISBON",
			ExpectedModeContains,
			501, //nolint:gomnd // magic number appropriate for test
		),
		GetJobSpec: func() executor.JobSpecDocker {
			return executor.JobSpecDocker{
				Image: "ubuntu:latest",
				Entrypoint: []string{
					"awk",
					"-F,",
					"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
					SimpleMountPath,
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
