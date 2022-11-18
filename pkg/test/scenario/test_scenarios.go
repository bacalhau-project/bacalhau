package scenario

import (
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

const HelloWorld = "hello world"
const SimpleMountPath = "/data/file.txt"
const SimpleOutputPath = "/output_data/output_file.txt"
const stdoutString = ipfs.DownloadFilenameStdout
const CatProgram = "cat " + SimpleMountPath + " > " + SimpleOutputPath

var CatFileToStdout = Scenario{
	Name: "cat_file_to_stdout",
	Inputs: StoredText(
		HelloWorld,
		SimpleMountPath,
	),
	Contexts: StoredFile(
		"../../../testdata/wasm/cat/main.wasm",
		"/job",
	),
	ResultsChecker: ManyChecks(
		FileEquals(ipfs.DownloadFilenameStderr, ""),
		FileEquals(ipfs.DownloadFilenameStdout, HelloWorld),
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
			Parameters: []string{SimpleMountPath},
		},
	},
}

var CatFileToVolume = Scenario{
	Name: "cat_file_to_volume",
	Inputs: StoredText(
		CatProgram,
		SimpleMountPath,
	),
	ResultsChecker: FileEquals(
		"test/output_file.txt",
		CatProgram,
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
				SimpleMountPath,
			},
		},
	},
}

var GrepFile = Scenario{
	Name: "grep_file",
	Inputs: StoredFile(
		"../../../testdata/grep_file.txt",
		SimpleMountPath,
	),
	ResultsChecker: FileContains(
		stdoutString,
		"kiwi is delicious",
		2,
	),
	Spec: model.Spec{
		Engine: model.EngineDocker,
		Docker: model.JobSpecDocker{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"grep",
				"kiwi",
				SimpleMountPath,
			},
		},
	},
}

var SedFile = Scenario{
	Name: "sed_file",
	Inputs: StoredFile(
		"../../../testdata/sed_file.txt",
		SimpleMountPath,
	),
	ResultsChecker: FileContains(
		stdoutString,
		"LISBON",
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
				SimpleMountPath,
			},
		},
	},
}

var AwkFile = Scenario{
	Name: "awk_file",
	Inputs: StoredFile(
		"../../../testdata/awk_file.txt",
		SimpleMountPath,
	),
	ResultsChecker: FileContains(
		stdoutString,
		"LISBON",
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
				SimpleMountPath,
			},
		},
	},
}

var WasmHelloWorld = Scenario{
	Name: "wasm_hello_world",
	Contexts: StoredFile(
		"../../../testdata/wasm/noop/main.wasm",
		"/job",
	),
	ResultsChecker: FileEquals(
		stdoutString,
		"Hello, world!\n",
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
			Parameters: []string{},
		},
	},
}

var WasmEnvVars = Scenario{
	Name: "wasm_env_vars",
	Contexts: StoredFile(
		"../../../testdata/wasm/env/main.wasm",
		"/job",
	),
	ResultsChecker: FileContains(
		"stdout",
		"AWESOME=definitely\nTEST=yes\n",
		3, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
			EnvironmentVariables: map[string]string{
				"TEST":    "yes",
				"AWESOME": "definitely",
			},
		},
	},
}

var WasmCsvTransform = Scenario{
	Name: "wasm_csv_transform",
	Inputs: StoredFile(
		"../../../testdata/wasm/csv/inputs",
		"/inputs",
	),
	Contexts: StoredFile(
		"../../../testdata/wasm/csv/main.wasm",
		"/job",
	),
	ResultsChecker: FileContains(
		"outputs/parents-children.csv",
		"http://www.wikidata.org/entity/Q14949904,Tugela,http://www.wikidata.org/entity/Q1001792,Makybe Diva",
		269, //nolint:gomnd // magic number appropriate for test
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
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

func GetAllScenarios() []Scenario {
	return []Scenario{
		CatFileToStdout,
		CatFileToVolume,
		GrepFile,
		SedFile,
		AwkFile,
		WasmHelloWorld,
		WasmEnvVars,
		WasmCsvTransform,
	}
}
