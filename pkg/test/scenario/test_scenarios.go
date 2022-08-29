package scenario

// TODO @enricorotundo #493: check if senarios can be move to testing tooling package

import (
	"context"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

const HelloWorld = "hello world"
const SimpleMountPath = "/data/file.txt"
const SimpleOutputPath = "/output_data/output_file.txt"
const stdoutString = "stdout"
const CatProgram = "cat " + SimpleMountPath + " > " + SimpleOutputPath

func CatFileToStdout(t *testing.T) TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "cat_file_to_stdout",
		SetupStorage: singleFileSetupStorageWithData(
			t,
			ctx,
			HelloWorld,
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			ctx,
			stdoutString,
			HelloWorld,
			ExpectedModeEquals,
			1,
		),
		GetJobSpec: func() model.JobSpecDocker {
			return model.JobSpecDocker{
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
	ctx := context.Background()
	return TestCase{
		Name: "cat_file_to_volume",
		SetupStorage: singleFileSetupStorageWithData(
			t,
			ctx,
			CatProgram,
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			ctx,
			"test/output_file.txt",
			CatProgram,
			ExpectedModeEquals,
			1,
		),
		Outputs: []model.StorageSpec{
			{
				Name: "test",
				Path: "/output_data",
			},
		},
		GetJobSpec: func() model.JobSpecDocker {
			return model.JobSpecDocker{
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
	ctx := context.Background()
	return TestCase{
		Name: "grep_file",
		SetupStorage: singleFileSetupStorageWithFile(
			t,
			ctx,
			"../../../testdata/grep_file.txt",
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			ctx,
			stdoutString,
			"kiwi is delicious",
			ExpectedModeContains,
			2,
		),
		GetJobSpec: func() model.JobSpecDocker {
			return model.JobSpecDocker{
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
	ctx := context.Background()
	return TestCase{
		Name: "sed_file",
		SetupStorage: singleFileSetupStorageWithFile(
			t,
			ctx,
			"../../../testdata/sed_file.txt",
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			ctx,
			stdoutString,
			"LISBON",
			ExpectedModeContains,
			5, //nolint:gomnd // magic number ok for testing
		),
		GetJobSpec: func() model.JobSpecDocker {
			return model.JobSpecDocker{
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
	ctx := context.Background()
	return TestCase{
		Name: "awk_file",
		SetupStorage: singleFileSetupStorageWithFile(
			t,
			ctx,
			"../../../testdata/awk_file.txt",
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			t,
			ctx,
			stdoutString,
			"LISBON",
			ExpectedModeContains,
			501, //nolint:gomnd // magic number appropriate for test
		),
		GetJobSpec: func() model.JobSpecDocker {
			return model.JobSpecDocker{
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
