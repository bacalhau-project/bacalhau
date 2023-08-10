package scenario

import (
	"runtime"
	"testing"

	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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

func CatFileToStdout(t testing.TB) Scenario {
	return Scenario{
		Inputs: StoredText(
			helloWorld,
			simpleMountPath,
		),
		ResultsChecker: ManyChecks(
			FileEquals(model.DownloadFilenameStderr, ""),
			FileEquals(model.DownloadFilenameStdout, helloWorld),
		),
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(InlineData(cat.Program())).
					WithEntrypoint("_start").
					WithParameters(simpleMountPath).
					Build(),
			),
		),
	}
}

func CatFileToVolume(t testing.TB) Scenario {
	return Scenario{
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
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewDockerEngineBuilder("ubuntu:latest").
					WithEntrypoint("bash", simpleMountPath).
					Build(),
			),
		),
	}
}

func GrepFile(t testing.TB) Scenario {
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
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewDockerEngineBuilder("ubuntu:latest").
					WithEntrypoint("grep", "kiwi", simpleMountPath).
					Build(),
			),
		),
	}
}

func SedFile(t testing.TB) Scenario {
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
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewDockerEngineBuilder("ubuntu:latest").
					WithEntrypoint(
						"sed",
						"-n",
						"/38.7[2-4]..,-9.1[3-7]../p",
						simpleMountPath,
					).
					Build(),
			),
		),
	}
}

func AwkFile(t testing.TB) Scenario {
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
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewDockerEngineBuilder("ubuntu:latest").
					WithEntrypoint(
						"awk",
						"-F,",
						"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
						simpleMountPath,
					).
					Build(),
			),
		),
	}
}

func WasmHelloWorld(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileEquals(
			model.DownloadFilenameStdout,
			"Hello, world!\n",
		),
		Spec: model.Spec{
			EngineSpec: model.NewWasmEngineBuilder(InlineData(noop.Program())).
				WithEntrypoint("_start").
				Build(),
		},
	}
}

func WasmExitCode(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileEquals(
			model.DownloadFilenameExitCode,
			"5",
		),
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(InlineData(exit_code.Program())).
					WithEntrypoint("_start").
					WithEnvironmentVariables(map[string]string{"EXIT_CODE": "5"}).
					Build(),
			),
		),
	}
}

func WasmEnvVars(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileContains(
			"stdout",
			[]string{"AWESOME=definitely", "TEST=yes"},
			3, //nolint:gomnd // magic number appropriate for test
		),
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(InlineData(env.Program())).
					WithEntrypoint("_start").
					WithEnvironmentVariables(
						map[string]string{
							"TEST":    "yes",
							"AWESOME": "definitely",
						},
					).
					Build(),
			),
		),
	}
}

func WasmCsvTransform(t testing.TB) Scenario {
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
		Outputs: []model.StorageSpec{
			{
				Name: "outputs",
				Path: "/outputs",
			},
		},
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(InlineData(csv.Program())).
					WithEntrypoint("_start").
					WithParameters(
						"inputs/horses.csv",
						"outputs/parents-children.csv",
					).
					Build(),
			),
		),
	}
}

func WasmDynamicLink(t testing.TB) Scenario {
	return Scenario{
		Inputs: StoredFile(
			"../../../testdata/wasm/easter/main.wasm",
			"/inputs",
		),
		ResultsChecker: FileEquals(
			model.DownloadFilenameStdout,
			"17\n",
		),
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(InlineData(dynamic.Program())).
					WithEntrypoint("_start").
					Build(),
			),
		),
	}
}

func WasmLogTest(t testing.TB) Scenario {
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
		Spec: testutils.MakeSpecWithOpts(t,
			jobutils.WithEngineSpec(
				model.NewWasmEngineBuilder(InlineData(logtest.Program())).
					WithEntrypoint("_start").
					WithParameters(
						"inputs/cosmic_computer.txt",
						"--fast",
					).
					Build(),
			),
		),
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
