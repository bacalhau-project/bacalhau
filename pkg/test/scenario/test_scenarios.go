package scenario

import (
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

const helloWorld = "hello world"
const simpleMountPath = "/data/file.txt"
const simpleOutputPath = "/output_data/output_file.txt"
const stdoutString = ipfs.DownloadFilenameStdout
const catProgram = "cat " + simpleMountPath + " > " + simpleOutputPath

var CatFileToStdout = Scenario{
	Inputs: StoredText(
		helloWorld,
		simpleMountPath,
	),
	Contexts: StoredFile(
		"../../../testdata/wasm/cat/main.wasm",
		"/job",
	),
	ResultsChecker: ManyChecks(
		FileEquals(ipfs.DownloadFilenameStderr, ""),
		FileEquals(ipfs.DownloadFilenameStdout, helloWorld),
	),
	Spec: model.Spec{
		Engine: model.EngineWasm,
		Wasm: model.JobSpecWasm{
			EntryPoint: "_start",
			Parameters: []string{simpleMountPath},
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
				simpleMountPath,
			},
		},
	},
}

var WasmHelloWorld = Scenario{
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

func GetAllScenarios() map[string]Scenario {
	return map[string]Scenario{
		"cat_file_to_stdout": CatFileToStdout,
		"cat_file_to_volume": CatFileToVolume,
		"grep_file":          GrepFile,
		"sed_file":           SedFile,
		"awk_file":           AwkFile,
		"wasm_hello_world":   WasmHelloWorld,
		"wasm_env_vars":      WasmEnvVars,
		"wasm_csv_transform": WasmCsvTransform,
	}
}
