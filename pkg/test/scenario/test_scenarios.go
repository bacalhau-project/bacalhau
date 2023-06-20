package scenario

import (
	"runtime"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/csv"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/dynamic"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/env"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/exit_code"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/logtest"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/noop"
)

const helloWorld = "hello world"
const simpleMountPath = "/data/file.txt"
const simpleOutputPath = "/output_data/output_file.txt"
const catProgram = "cat " + simpleMountPath + " > " + simpleOutputPath

var CatFileToStdout = Scenario{
	Inputs: StoredText(
		helloWorld,
		simpleMountPath,
	),
	ResultsChecker: ManyChecks(
		FileEquals(model.DownloadFilenameStderr, ""),
		FileEquals(model.DownloadFilenameStdout, helloWorld),
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(cat.Program()),
			Parameters:  []string{simpleMountPath},
		},
	},
}

var CatFileToVolume = Scenario{
	Inputs: StoredText(
		catProgram,
		simpleMountPath,
	),
	ResultsChecker: FileEquals(
		"test/output_file.txt",
		catProgram,
	),
	Outputs: []model.StorageSpec{
		{
			Name: "test",
			Path: "/output_data",
		},
	},
	Spec: model.Spec{
		Engine: model.EngineDocker,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash",
				simpleMountPath,
			},
		},
	},
}

var GrepFile = Scenario{
	Inputs: StoredFile(
		"../../../testdata/grep_file.txt",
		simpleMountPath,
	),
	ResultsChecker: FileContains(
		model.DownloadFilenameStdout,
		[]string{"kiwi is delicious"},
		2,
	),
	Spec: model.Spec{
		Engine: model.EngineDocker,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"grep",
				"kiwi",
				simpleMountPath,
			},
		},
	},
}

var SedFile = Scenario{
	Inputs: StoredFile(
		"../../../testdata/sed_file.txt",
		simpleMountPath,
	),
	ResultsChecker: FileContains(
		model.DownloadFilenameStdout,
		[]string{"LISBON"},
		5, //nolint:gomnd // magic number ok for testing
	),
	Spec: model.Spec{
		Engine: model.EngineDocker,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"sed",
				"-n",
				"/38.7[2-4]..,-9.1[3-7]../p",
				simpleMountPath,
			},
		},
	},
}

var AwkFile = Scenario{
	Inputs: StoredFile(
		"../../../testdata/awk_file.txt",
		simpleMountPath,
	),
	ResultsChecker: FileContains(
		model.DownloadFilenameStdout,
		[]string{"LISBON"},
		501, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		Engine: model.EngineDocker,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"awk",
				"-F,",
				"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
				simpleMountPath,
			},
		},
	},
}

var WasmHelloWorld = Scenario{
	ResultsChecker: FileEquals(
		model.DownloadFilenameStdout,
		"Hello, world!\n",
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(noop.Program()),
			Parameters:  []string{},
		},
	},
}

var WasmExitCode = Scenario{
	ResultsChecker: FileEquals(
		model.DownloadFilenameExitCode,
		"5",
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:           "_start",
			EntryModule:          InlineData(exit_code.Program()),
			Parameters:           []string{},
			EnvironmentVariables: map[string]string{"EXIT_CODE": "5"},
		},
	},
}

var WasmEnvVars = Scenario{
	ResultsChecker: FileContains(
		"stdout",
		[]string{"AWESOME=definitely", "TEST=yes"},
		3, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(env.Program()),
			EnvironmentVariables: map[string]string{
				"TEST":    "yes",
				"AWESOME": "definitely",
			},
		},
	},
}

var WasmCsvTransform = Scenario{
	Inputs: StoredFile(
		"../../../testdata/wasm/csv/inputs",
		"/inputs",
	),
	ResultsChecker: FileContains(
		"outputs/parents-children.csv",
		[]string{"http://www.wikidata.org/entity/Q14949904,Tugela,http://www.wikidata.org/entity/Q1001792,Makybe Diva"},
		269, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(csv.Program()),
			Parameters: []string{
				"inputs/horses.csv",
				"outputs/parents-children.csv",
			},
		},
	},
	Outputs: []model.StorageSpec{
		{
			Name: "outputs",
			Path: "/outputs",
		},
	},
}

var WasmDynamicLink = Scenario{
	Inputs: StoredFile(
		"../../../testdata/wasm/easter/main.wasm",
		"/inputs",
	),
	ResultsChecker: FileEquals(
		model.DownloadFilenameStdout,
		"17\n",
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(dynamic.Program()),
		},
	},
}

var WasmLogTest = Scenario{
	Inputs: StoredFile(
		"../../../testdata/wasm/logtest/inputs/",
		"/inputs",
	),
	ResultsChecker: FileContains(
		"stdout",
		[]string{"https://www.gutenberg.org"}, // end of the file
		-1,                                    //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(logtest.Program()),
			Parameters: []string{
				"inputs/cosmic_computer.txt",
				"--fast",
			},
		},
	},
}

func GetAllScenarios() map[string]Scenario {
	scenarios := map[string]Scenario{
		"cat_file_to_stdout": CatFileToStdout,
		"cat_file_to_volume": CatFileToVolume,
		"grep_file":          GrepFile,
		"sed_file":           SedFile,
		"awk_file":           AwkFile,
		"logtest":            WasmLogTest,
		"wasm_hello_world":   WasmHelloWorld,
		"wasm_env_vars":      WasmEnvVars,
		"wasm_csv_transform": WasmCsvTransform,
		"wasm_exit_code":     WasmExitCode,
		"wasm_dynamic_link":  WasmDynamicLink,
	}

	if runtime.GOOS == "windows" {
		// Temporarily skip the wasm_env_vars test on windows to avoid
		// flakiness until we can resolve the problem.
		delete(scenarios, "wasm_env_vars")
	}

	return scenarios
}
