package scenario

import (
	"context"
	"fmt"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

const HelloWorld = "hello world"
const SimpleMountPath = "/data/file.txt"
const SimpleOutputPath = "/output_data/output_file.txt"
const stdoutString = "stdout"
const CatProgram = "cat " + SimpleMountPath + " > " + SimpleOutputPath

func Sleep(sleepSeconds float32) TestCase {
	sleepSecondsStr := fmt.Sprintf("%.3f", sleepSeconds)

	return TestCase{
		Name: "sleep_" + sleepSecondsStr + "_seconds",
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineDocker,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"sleep",
						sleepSecondsStr,
					},
				},
			}
		},
	}
}

func CatFileToStdout() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "cat_file_to_stdout",
		SetupStorage: singleFileSetupStorageWithData(
			ctx,
			HelloWorld,
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			stdoutString,
			HelloWorld,
			ExpectedModeEquals,
			1,
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineDocker,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"cat",
						SimpleMountPath,
					},
				},
			}
		},
	}
}

func CatFileToVolume() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "cat_file_to_volume",
		SetupStorage: singleFileSetupStorageWithData(
			ctx,
			CatProgram,
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
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
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineDocker,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"bash",
						SimpleMountPath,
					},
				},
			}
		},
	}
}

func GrepFile() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "grep_file",
		SetupStorage: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/grep_file.txt",
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			stdoutString,
			"kiwi is delicious",
			ExpectedModeContains,
			2,
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineDocker,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"grep",
						"kiwi",
						SimpleMountPath,
					},
				},
			}
		},
	}
}

func SedFile() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "sed_file",
		SetupStorage: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/sed_file.txt",
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			stdoutString,
			"LISBON",
			ExpectedModeContains,
			5, //nolint:gomnd // magic number ok for testing
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineDocker,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"sed",
						"-n",
						"/38.7[2-4]..,-9.1[3-7]../p",
						SimpleMountPath,
					},
				},
			}
		},
	}
}

func AwkFile() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "awk_file",
		SetupStorage: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/awk_file.txt",
			SimpleMountPath,
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			stdoutString,
			"LISBON",
			ExpectedModeContains,
			501, //nolint:gomnd // magic number appropriate for test
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineDocker,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"awk",
						"-F,",
						"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
						SimpleMountPath,
					},
				},
			}
		},
	}
}

func WasmHelloWorld() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "wasm_hello_world",
		SetupContext: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/wasm/noop/main.wasm",
			"/job",
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			stdoutString,
			"Hello, world!\n",
			ExpectedModeEquals,
			2, //nolint:gomnd // magic number appropriate for test
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineWasm,
				Wasm: model.JobSpecWasm{
					EntryPoint: "_start",
					Parameters: []string{},
				},
			}
		},
	}
}

func WasmEnvVars() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "wasm_env_vars",
		SetupContext: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/wasm/env/main.wasm",
			"/job",
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			"stdout",
			"AWESOME=definitely\nTEST=yes\n",
			ExpectedModeEquals,
			3, //nolint:gomnd // magic number appropriate for test
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineWasm,
				Wasm: model.JobSpecWasm{
					EntryPoint: "_start",
					EnvironmentVariables: map[string]string{
						"TEST":    "yes",
						"AWESOME": "definitely",
					},
				},
			}
		},
	}
}

func WasmCsvTransform() TestCase {
	ctx := context.Background()
	return TestCase{
		Name: "wasm_csv_transform",
		SetupStorage: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/wasm/csv/inputs",
			"/inputs",
		),
		SetupContext: singleFileSetupStorageWithFile(
			ctx,
			"../../../testdata/wasm/csv/main.wasm",
			"/job",
		),
		ResultsChecker: singleFileResultsChecker(
			ctx,
			"outputs/parents-children.csv",
			"http://www.wikidata.org/entity/Q14949904,Tugela,http://www.wikidata.org/entity/Q1001792,Makybe Diva",
			ExpectedModeContains,
			269, //nolint:gomnd // magic number appropriate for test
		),
		GetJobSpec: func() model.Spec {
			return model.Spec{
				Engine: model.EngineWasm,
				Wasm: model.JobSpecWasm{
					EntryPoint: "_start",
					Parameters: []string{
						"inputs/horses.csv",
						"outputs/parents-children.csv",
					},
				},
			}
		},
		Outputs: []model.StorageSpec{
			{
				Name: "outputs",
				Path: "/outputs",
			},
		},
	}
}

func GetAllScenarios() []TestCase {
	return []TestCase{
		CatFileToStdout(),
		CatFileToVolume(),
		GrepFile(),
		SedFile(),
		AwkFile(),
		WasmHelloWorld(),
		WasmEnvVars(),
		WasmCsvTransform(),
	}
}
