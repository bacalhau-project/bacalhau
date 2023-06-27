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
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(cat.Program()), "_start", []string{simpleMountPath}, nil, nil),
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
		EngineDeprecated: model.EngineDocker,
		EngineSpec:       model.NewDockerEngineSpec("ubuntu:latest", []string{"bash", simpleMountPath}, nil, ""),
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
		EngineDeprecated: model.EngineDocker,
		EngineSpec:       model.NewDockerEngineSpec("ubuntu:latest", []string{"grep", "kiwi", simpleMountPath}, nil, ""),
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
		EngineDeprecated: model.EngineDocker,
		EngineSpec: model.NewDockerEngineSpec("ubuntu:latest", []string{
			"sed",
			"-n",
			"/38.7[2-4]..,-9.1[3-7]../p",
			simpleMountPath,
		}, nil, ""),
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
		EngineDeprecated: model.EngineDocker,
		EngineSpec: model.NewDockerEngineSpec("ubuntu:latest", []string{
			"awk",
			"-F,",
			"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
			simpleMountPath,
		}, nil, ""),
	},
}

var WasmHelloWorld = Scenario{
	ResultsChecker: FileEquals(
		model.DownloadFilenameStdout,
		"Hello, world!\n",
	),
	Spec: model.Spec{
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(noop.Program()), "_start", []string{}, nil, nil),
	},
}

var WasmExitCode = Scenario{
	ResultsChecker: FileEquals(
		model.DownloadFilenameExitCode,
		"5",
	),
	Spec: model.Spec{
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(exit_code.Program()), "_start", []string{}, map[string]string{"EXIT_CODE": "5"}, nil),
	},
}

var WasmEnvVars = Scenario{
	ResultsChecker: FileContains(
		"stdout",
		[]string{"AWESOME=definitely", "TEST=yes"},
		3, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(env.Program()), "_start", nil, map[string]string{"TEST": "yes", "AWESOME": "definitely"}, nil),
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
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(csv.Program()), "_start", []string{"inputs/horses.csv", "outputs/parents-children.csv"}, nil, nil),
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
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(dynamic.Program()), "_start", nil, nil, nil),
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
		EngineDeprecated: model.EngineWasm,
		EngineSpec:       model.NewWasmEngineSpec(InlineData(logtest.Program()), "_start", []string{"inputs/cosmic_computer.txt", "--fast"}, nil, nil),
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
