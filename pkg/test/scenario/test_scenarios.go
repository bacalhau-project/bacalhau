package scenario

import (
	"os"
	"runtime"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	http "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/funcs/http/testdata"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
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
const defaultDockerImage = "busybox:1.37.0"

const AllowedListedLocalPathsSuffix = string(os.PathSeparator) + "*"

func CatFileToStdout(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredText(
			rootSourceDir,
			helloWorld,
			simpleMountPath,
		),
		ResultsChecker: ManyChecks(
			FileEquals(downloader.DownloadFilenameStderr, ""),
			FileEquals(downloader.DownloadFilenameStdout, helloWorld),
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					InputSources: []*models.InputSource{
						InlineDataWithTarget(cat.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						WithParameters(simpleMountPath).MustBuild(),
				},
			},
		},
	}
}

func CatFileToVolume(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredText(
			rootSourceDir,
			catProgram,
			simpleMountPath,
		),
		ResultsChecker: FileEquals(
			"test/output_file.txt",
			catProgram,
		),
		Outputs: []*models.ResultPath{
			{
				Name: "test",
				Path: "/output_data",
			},
		},
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: dockmodels.NewDockerEngineBuilder(defaultDockerImage).
						WithEntrypoint("sh", simpleMountPath).MustBuild(),
				},
			},
		},
	}
}

func GrepFile(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredFile(
			rootSourceDir,
			"../../../testdata/grep_file.txt",
			simpleMountPath,
		),
		ResultsChecker: FileContains(
			downloader.DownloadFilenameStdout,
			[]string{"kiwi is delicious"},
			2,
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: dockmodels.NewDockerEngineBuilder(defaultDockerImage).
						WithEntrypoint("grep", "kiwi", simpleMountPath).
						MustBuild(),
				},
			},
		},
	}
}

func SedFile(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredFile(
			rootSourceDir,
			"../../../testdata/sed_file.txt",
			simpleMountPath,
		),
		ResultsChecker: FileContains(
			downloader.DownloadFilenameStdout,
			[]string{"LISBON"},
			5, //nolint:mnd // magic number ok for testing
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: dockmodels.NewDockerEngineBuilder(defaultDockerImage).
						WithEntrypoint("sed", "-n", "/38.7[2-4]..,-9.1[3-7]../p", simpleMountPath).
						MustBuild(),
				},
			},
		},
	}
}

func AwkFile(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredFile(
			rootSourceDir,
			"../../../testdata/awk_file.txt",
			simpleMountPath,
		),
		ResultsChecker: FileContains(
			downloader.DownloadFilenameStdout,
			[]string{"LISBON"},
			501, //nolint:mnd // magic number appropriate for test
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Engine: dockmodels.NewDockerEngineBuilder(defaultDockerImage).
						WithEntrypoint(
							"awk",
							"-F,",
							"{x=38.7077507-$3; y=-9.1365919-$4; if(x^2+y^2<0.3^2) print}",
							simpleMountPath,
						).
						MustBuild(),
				},
			},
		},
	}
}

func WasmHelloWorld(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileEquals(
			downloader.DownloadFilenameStdout,
			"Hello, world!\n",
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					InputSources: []*models.InputSource{
						InlineDataWithTarget(noop.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
	}
}

func WasmExitCode(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileEquals(
			downloader.DownloadFilenameExitCode,
			"5",
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Env:  map[string]models.EnvVarValue{"EXIT_CODE": "5"},
					InputSources: []*models.InputSource{
						InlineDataWithTarget(exit_code.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
				},
			},
		},
	}
}

func WasmEnvVars(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileContains(
			"stdout",
			[]string{"AWESOME=definitely", "TEST=yes", "BACALHAU_PARTITION_INDEX=0"},
			-1,
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Env:  map[string]models.EnvVarValue{"TEST": "yes", "AWESOME": "definitely"},
					InputSources: []*models.InputSource{
						InlineDataWithTarget(env.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
				},
			},
		},
	}
}

func WasmCsvTransform(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredFile(
			rootSourceDir,
			"../../../testdata/wasm/csv/inputs",
			"/inputs",
		),
		ResultsChecker: FileContains(
			"outputs/parents-children.csv",
			[]string{"http://www.wikidata.org/entity/Q14949904,Tugela,http://www.wikidata.org/entity/Q1001792,Makybe Diva"},
			269, //nolint:mnd // magic number appropriate for test
		),
		Outputs: []*models.ResultPath{
			{
				Name: "outputs",
				Path: "/outputs",
			},
		},
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					InputSources: []*models.InputSource{
						InlineDataWithTarget(csv.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						WithParameters(
							"inputs/horses.csv",
							"outputs/parents-children.csv",
						).
						MustBuild(),
				},
			},
		},
	}
}

func WasmDynamicLink(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: ManyStores(
			StoredText(rootSourceDir, "unused input", "/data"),

			// We are mounting/aliasing the wasm file as a input.wasm as this is what dynamic.wasm expects.
			StoredFile(rootSourceDir,
				"../../../testdata/wasm/easter/main.wasm",
				"input.wasm",
			),
		),
		ResultsChecker: FileEquals(
			downloader.DownloadFilenameStdout,
			"17\n",
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					InputSources: []*models.InputSource{
						InlineDataWithTarget(dynamic.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
				},
			},
		},
	}
}

func WasmLogTest(t testing.TB) Scenario {
	rootSourceDir := t.TempDir()

	return Scenario{
		Stack: &StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + AllowedListedLocalPathsSuffix}),
			},
		},
		Inputs: StoredFile(rootSourceDir,
			"../../../testdata/wasm/logtest/inputs/",
			"/inputs",
		),
		ResultsChecker: FileContains(
			"stdout",
			[]string{"https://www.gutenberg.org"}, // end of the file
			-1,
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					InputSources: []*models.InputSource{
						InlineDataWithTarget(logtest.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						WithParameters(
							"inputs/cosmic_computer.txt",
							"--fast",
						).
						MustBuild(),
				},
			},
		},
	}
}

func WasmGetHTTP(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileContains(
			"stdout",
			[]string{"Status Code: 200"},
			-1,
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Env:  map[string]models.EnvVarValue{"HTTP_METHOD": "get", "HTTP_URL": "http://google.com/"},
					InputSources: []*models.InputSource{
						InlineDataWithTarget(http.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
					Network: &models.NetworkConfig{
						Type: models.NetworkFull,
					},
				},
			},
		},
	}
}

func WasmGetHTTPAllowList(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileContains(
			"stdout",
			[]string{"Status Code: 200"},
			-1,
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Env:  map[string]models.EnvVarValue{"HTTP_METHOD": "get", "HTTP_URL": "http://google.com/"},
					InputSources: []*models.InputSource{
						InlineDataWithTarget(http.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
					Network: &models.NetworkConfig{
						Type:    models.NetworkHTTP,
						Domains: []string{"google.com"},
					},
				},
			},
		},
	}
}

func WasmGetHTTPNotAllowList(t testing.TB) Scenario {
	return Scenario{
		ResultsChecker: FileContains(
			"stdout",
			[]string{"host not allowed"},
			-1,
		),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Env:  map[string]models.EnvVarValue{"HTTP_METHOD": "get", "HTTP_URL": "http://google.com/"},
					InputSources: []*models.InputSource{
						InlineDataWithTarget(http.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
					Network: &models.NetworkConfig{
						Type:    models.NetworkHTTP,
						Domains: []string{"httpbin.com"},
					},
				},
			},
		},
	}
}

func WasmNoNetworking(t testing.TB) Scenario {
	return Scenario{
		CommandResultsChecker: ErrorMessageContains(`unknown module "wasi:http/requests"`),
		Job: &models.Job{
			Name:  t.Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: t.Name(),
					Env:  map[string]models.EnvVarValue{"HTTP_METHOD": "get", "HTTP_URL": "http://google.com/"},
					InputSources: []*models.InputSource{
						InlineDataWithTarget(http.Program(), "main.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("main.wasm").
						WithEntrypoint("_start").
						MustBuild(),
					Network: &models.NetworkConfig{
						Type: models.NetworkNone,
					},
				},
			},
		},
	}
}

func GetAllScenarios(t testing.TB) map[string]Scenario {
	scenarios := map[string]Scenario{
		"cat_file_to_stdout":        CatFileToStdout(t),
		"cat_file_to_volume":        CatFileToVolume(t),
		"grep_file":                 GrepFile(t),
		"sed_file":                  SedFile(t),
		"awk_file":                  AwkFile(t),
		"logtest":                   WasmLogTest(t),
		"wasm_hello_world":          WasmHelloWorld(t),
		"wasm_env_vars":             WasmEnvVars(t),
		"wasm_csv_transform":        WasmCsvTransform(t),
		"wasm_exit_code":            WasmExitCode(t),
		"wasm_dynamic_link":         WasmDynamicLink(t),
		"wasm_http_get_full":        WasmGetHTTP(t),
		"wasm_http_get_allowlist":   WasmGetHTTPAllowList(t),
		"wasm_http_not_allowlisted": WasmGetHTTPNotAllowList(t),
		"wasm_no_networking":        WasmNoNetworking(t),
	}

	if runtime.GOOS == "windows" {
		// Temporarily skip the wasm_env_vars test on windows to avoid
		// flakiness until we can resolve the problem.
		delete(scenarios, "wasm_env_vars")
	}

	return scenarios
}
