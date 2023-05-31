package scenario

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
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

var CatFileToStdout = func(t testing.TB) Scenario {
	return Scenario{
		Inputs: StoredText(
			helloWorld,
			simpleMountPath,
		),
		ResultsChecker: ManyChecks(
			FileEquals(model.DownloadFilenameStderr, ""),
			FileEquals(model.DownloadFilenameStdout, helloWorld),
		),
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(cat.Program())),
				enginetesting.WasmWithEntrypoint("_start"),
			),
		},
	}
}

var CatFileToVolume = func(t testing.TB) Scenario {
	localspec, err := (&local.LocalStorageSpec{Source: "CatFileToVolumeTest"}).AsSpec("test", "/output_data")
	require.NoError(t, err)
	return Scenario{
		Inputs: StoredText(
			catProgram,
			simpleMountPath,
		),
		ResultsChecker: FileEquals(
			"test/output_file.txt",
			catProgram,
		),
		Outputs: []spec.Storage{localspec},
		Spec: model.Spec{
			Engine: enginetesting.DockerMakeEngine(t,
				enginetesting.DockerWithImage("ubuntu:latest"),
				enginetesting.DockerWithEntrypoint("bash", simpleMountPath),
			),
		},
	}
}

var GrepFile = func(t testing.TB) Scenario {
	return Scenario{
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
			Engine: enginetesting.DockerMakeEngine(t,
				enginetesting.DockerWithImage("ubuntu:latest"),
				enginetesting.DockerWithEntrypoint("grep", "kiwi", simpleMountPath),
			),
		},
	}
}

var SedFile = func(t testing.TB) Scenario {
	return Scenario{
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
			Engine: enginetesting.DockerMakeEngine(t,
				enginetesting.DockerWithImage("ubuntu:latest"),
				enginetesting.DockerWithEntrypoint("sed", "-n", "/38.7[2-4]..,-9.1[3-7]../p", simpleMountPath),
			),
		},
	}
}

var AwkFile = func(t testing.TB) Scenario {
	return Scenario{
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
			Engine: enginetesting.DockerMakeEngine(t,
				enginetesting.DockerWithImage("ubuntu:latest"),
				enginetesting.DockerWithEntrypoint("awk", "-F,", "{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}", simpleMountPath),
			),
		},
	}
}

var WasmHelloWorld = func(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileEquals(
			model.DownloadFilenameStdout,
			"Hello, world!\n",
		),
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(noop.Program())),
				enginetesting.WasmWithParameters([]string{}...),
			),
		},
	}
}

var WasmExitCode = func(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileEquals(
			model.DownloadFilenameExitCode,
			"5",
		),
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(exit_code.Program())),
				enginetesting.WasmWithParameters([]string{}...),
				enginetesting.WasmWithEnvironmentVariables("EXIT_CODE", "5"),
			),
		},
	}
}

var WasmEnvVars = func(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileContains(
			"stdout",
			[]string{"AWESOME=definitely", "TEST=yes"},
			3, //nolint:gomnd // magic number appropriate for test
		),
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(env.Program())),
				enginetesting.WasmWithEnvironmentVariables("TEST", "yes", "AWESOME", "definitely"),
			),
		},
	}
}

var WasmCsvTransform = func(t testing.TB) Scenario {
	localspec, err := (&local.LocalStorageSpec{Source: "WasmCsvTransform"}).AsSpec("outputs", "/outputs")
	require.NoError(t, err)
	return Scenario{
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
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(csv.Program())),
				enginetesting.WasmWithParameters("inputs/horses.csv", "outputs/parents-children.csv"),
			),
		},
		Outputs: []spec.Storage{localspec},
	}
}

var WasmDynamicLink = func(t testing.TB) Scenario {
	return Scenario{
		Inputs: StoredFile(
			"../../../testdata/wasm/easter/main.wasm",
			"/inputs",
		),
		ResultsChecker: FileEquals(
			model.DownloadFilenameStdout,
			"17\n",
		),
		Spec: model.Spec{
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(dynamic.Program())),
			),
		},
	}
}

var WasmLogTest = func(t testing.TB) Scenario {
	return Scenario{
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
			Engine: enginetesting.WasmMakeEngine(t,
				enginetesting.WasmWithEntrypoint("_start"),
				enginetesting.WasmWithEntryModule(InlineData(logtest.Program())),
				enginetesting.WasmWithParameters("inputs/cosmic_computer.txt", "--fast"),
			),
		},
	}
}

func GetAllScenarios(t testing.TB) map[string]Scenario {
	scenarios := map[string]Scenario{
		"cat_file_to_stdout": CatFileToStdout(t),
		"cat_file_to_volume": CatFileToVolume(t),
		"grep_file":          GrepFile(t),
		"sed_file":           SedFile(t),
		"awk_file":           AwkFile(t),
		"logtest":            WasmLogTest(t),
		"wasm_hello_world":   WasmHelloWorld(t),
		"wasm_env_vars":      WasmEnvVars(t),
		"wasm_csv_transform": WasmCsvTransform(t),
		"wasm_exit_code":     WasmExitCode(t),
		"wasm_dynamic_link":  WasmDynamicLink(t),
	}

	if runtime.GOOS == "windows" {
		// Temporarily skip the wasm_env_vars test on windows to avoid
		// flakiness until we can resolve the problem.
		delete(scenarios, "wasm_env_vars")
	}

	return scenarios
}
