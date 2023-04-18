package scenario

import (
	docker_spec "github.com/bacalhau-project/bacalhau/pkg/executor/docker/spec"
	wasm_spec "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/spec"
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
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(cat.Program()),
			Parameters:  []string{simpleMountPath},
		}).AsEngineSpec(),
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
		EngineSpec: (&docker_spec.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"bash",
				simpleMountPath,
			},
		}).AsEngineSpec(),
	},
}

var GrepFile = Scenario{
	Inputs: StoredFile(
		"../../../testdata/grep_file.txt",
		simpleMountPath,
	),
	ResultsChecker: FileContains(
		model.DownloadFilenameStdout,
		"kiwi is delicious",
		2,
	),
	Spec: model.Spec{
		EngineSpec: (&docker_spec.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"grep",
				"kiwi",
				simpleMountPath,
			},
		}).AsEngineSpec(),
	},
}

var SedFile = Scenario{
	Inputs: StoredFile(
		"../../../testdata/sed_file.txt",
		simpleMountPath,
	),
	ResultsChecker: FileContains(
		model.DownloadFilenameStdout,
		"LISBON",
		5, //nolint:gomnd // magic number ok for testing
	),
	Spec: model.Spec{
		EngineSpec: (&docker_spec.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"sed",
				"-n",
				"/38.7[2-4]..,-9.1[3-7]../p",
				simpleMountPath,
			},
		}).AsEngineSpec(),
	},
}

var AwkFile = Scenario{
	Inputs: StoredFile(
		"../../../testdata/awk_file.txt",
		simpleMountPath,
	),
	ResultsChecker: FileContains(
		model.DownloadFilenameStdout,
		"LISBON",
		501, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		EngineSpec: (&docker_spec.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"awk",
				"-F,",
				"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
				simpleMountPath,
			},
		}).AsEngineSpec(),
	},
}

var WasmHelloWorld = Scenario{
	ResultsChecker: FileEquals(
		model.DownloadFilenameStdout,
		"Hello, world!\n",
	),
	Spec: model.Spec{
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(noop.Program()),
			Parameters:  []string{},
		}).AsEngineSpec(),
	},
}

var WasmExitCode = Scenario{
	ResultsChecker: FileEquals(
		model.DownloadFilenameExitCode,
		"5",
	),
	Spec: model.Spec{
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:           "_start",
			EntryModule:          InlineData(exit_code.Program()),
			Parameters:           []string{},
			EnvironmentVariables: map[string]string{"EXIT_CODE": "5"},
		}).AsEngineSpec(),
	},
}

var WasmEnvVars = Scenario{
	ResultsChecker: FileContains(
		"stdout",
		"AWESOME=definitely\nTEST=yes\n",
		3, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(env.Program()),
			EnvironmentVariables: map[string]string{
				"TEST":    "yes",
				"AWESOME": "definitely",
			},
		}).AsEngineSpec(),
	},
}

var WasmCsvTransform = Scenario{
	Inputs: StoredFile(
		"../../../testdata/wasm/csv/inputs",
		"/inputs",
	),
	ResultsChecker: FileContains(
		"outputs/parents-children.csv",
		"http://www.wikidata.org/entity/Q14949904,Tugela,http://www.wikidata.org/entity/Q1001792,Makybe Diva",
		269, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(csv.Program()),
			Parameters: []string{
				"inputs/horses.csv",
				"outputs/parents-children.csv",
			},
		}).AsEngineSpec(),
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
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(dynamic.Program()),
		}).AsEngineSpec(),
	},
}

var WasmLogTest = Scenario{
	Inputs: StoredFile(
		"../../../testdata/wasm/logtest/inputs/",
		"/inputs",
	),
	ResultsChecker: FileContains(
		"stdout",
		"https://www.gutenberg.org", // end of the file
		5216,                        //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		EngineSpec: (&wasm_spec.JobSpecWasm{
			EntryPoint:  "_start",
			EntryModule: InlineData(logtest.Program()),
			Parameters: []string{
				"inputs/cosmic_computer.txt",
				"--slow",
			},
		}).AsEngineSpec(),
	},
}

func GetAllScenarios() map[string]Scenario {
	return map[string]Scenario{
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
}
