//go:build integration || !unit

package devstack

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	publisher_local "github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type URLTestSuite struct {
	scenario.ScenarioRunner
}

func TestURLTests(t *testing.T) {
	suite.Run(t, new(URLTestSuite))
}

type URLBasedTestCase struct {
	file1  string
	file2  string
	mount1 string
	mount2 string
	files  map[string]string
}

func runURLTest(
	suite *URLTestSuite,
	handler func(w http.ResponseWriter, r *http.Request),
	testCase URLBasedTestCase,
) {
	svr := httptest.NewServer(http.HandlerFunc(handler))
	defer func() { _ = svr.Close() }()

	allContent := testCase.files[fmt.Sprintf("/%s", testCase.file1)] + testCase.files[fmt.Sprintf("/%s", testCase.file2)]
	testScenario := scenario.Scenario{
		Stack: &scenario.StackConfig{},
		Inputs: scenario.ManyStores(
			scenario.URLDownload(svr, testCase.file1, testCase.mount1),
			scenario.URLDownload(svr, testCase.file2, testCase.mount2),
		),
		ResultsChecker: scenario.ManyChecks(
			scenario.FileEquals(downloader.DownloadFilenameStderr, ""),
			scenario.FileEquals(downloader.DownloadFilenameStdout, allContent),
		),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
		},

		Job: &models.Job{
			Name:  suite.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: suite.T().Name(),
					InputSources: []*models.InputSource{
						scenario.InlineDataWithTarget(cat.Program(), "/app/cat.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("/app/cat.wasm").
						WithEntrypoint("_start").
						WithParameters(
							path.Join(testCase.mount1, testCase.file1),
							path.Join(testCase.mount2, testCase.file2),
						).MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
	}

	suite.RunScenario(testScenario)
}

func getSimpleTestCase() URLBasedTestCase {
	file1 := "hello-cid-1.txt"
	file2 := "hello-cid-2.txt"
	return URLBasedTestCase{
		file1:  file1,
		file2:  file2,
		mount1: "/inputs-1",
		mount2: "/inputs-2",
		files: map[string]string{
			fmt.Sprintf("/%s", file1): "Before you marry a person, you should first make them use a computer with slow Internet to see who they really are.\n",
			fmt.Sprintf("/%s", file2): "I walk around like everything's fine, but deep down, inside my shoe, my sock is sliding off.\n",
		},
	}
}

func (s *URLTestSuite) TestMultipleURLs() {
	testCase := getSimpleTestCase()
	handler := func(w http.ResponseWriter, r *http.Request) {
		content, ok := testCase.files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}
	}
	runURLTest(s, handler, testCase)
}

// both starts should be before both ends if we are downloading in parallel
func (s *URLTestSuite) TestURLsInParallel() {
	mutex := sync.Mutex{}
	testCase := getSimpleTestCase()

	accessTimes := map[string]int64{}
	getAccessTime := func() int64 {
		return time.Now().UnixNano() / int64(time.Millisecond)
	}
	getAccessKey := func(filename, append string) string {
		return fmt.Sprintf("%s_%s", filename, append)
	}
	setAccessTime := func(key string) {
		mutex.Lock()
		defer mutex.Unlock()
		accessTimes[key] = getAccessTime()
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		setAccessTime(getAccessKey(r.URL.Path, "start"))
		time.Sleep(time.Second * 1)
		setAccessTime(getAccessKey(r.URL.Path, "end"))
		content, ok := testCase.files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}

	}
	runURLTest(s, handler, testCase)

	start1, ok := accessTimes["/"+getAccessKey(testCase.file1, "start")]
	require.True(s.T(), ok)
	start2, ok := accessTimes["/"+getAccessKey(testCase.file2, "start")]
	require.True(s.T(), ok)
	end1, ok := accessTimes["/"+getAccessKey(testCase.file1, "end")]
	require.True(s.T(), ok)
	end2, ok := accessTimes["/"+getAccessKey(testCase.file2, "end")]
	require.True(s.T(), ok)

	require.True(s.T(), start2 < end1, "start 2 should be before end 1")
	require.True(s.T(), start1 < end2, "start 1 should be before end 2")
}

func (s *URLTestSuite) TestFlakyURLs() {
	mutex := sync.Mutex{}
	testCase := getSimpleTestCase()
	accessCounter := map[string]int{}
	increaseCounter := func(key string) int {
		mutex.Lock()
		defer mutex.Unlock()
		accessCount, ok := accessCounter[key]
		if !ok {
			accessCount = 0
		}
		accessCount++
		accessCounter[key] = accessCount
		return accessCount
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		accessCounts := increaseCounter(r.URL.Path)
		if accessCounts < config.Default.InputSources.MaxRetryCount {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("not found"))
			return
		}
		content, ok := testCase.files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}

	}
	runURLTest(s, handler, testCase)
}

func (s *URLTestSuite) TestLocalURLCombo() {
	localFile := "hello-local.txt"
	urlfile := "hello-url.txt"
	localMount := "/inputs-1"
	urlmount := "/inputs-2"

	URLContent := "Common sense is like deodorant. The people who need it most never use it.\n"
	localContent := "Truth hurts. Maybe not as much as jumping on a bicycle with a seat missing, but it hurts.\n"

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(URLContent))
	}))
	defer func() { _ = svr.Close() }()

	rootSourceDir := s.T().TempDir()
	testScenario := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithAllowListedLocalPaths([]string{rootSourceDir + scenario.AllowedListedLocalPathsSuffix}),
				devstack.WithBacalhauConfigOverride(types.Bacalhau{
					JobAdmissionControl: types.JobAdmissionControl{
						Locality: models.Anywhere,
					},
				}),
			},
		},
		Inputs: scenario.ManyStores(
			scenario.StoredText(rootSourceDir, localContent, path.Join(localMount, localFile)),
			scenario.URLDownload(svr, urlfile, urlmount),
		),
		Job: &models.Job{
			Name:  s.T().Name(),
			Type:  models.JobTypeBatch,
			Count: 1,
			Tasks: []*models.Task{
				{
					Name: s.T().Name(),
					InputSources: []*models.InputSource{
						scenario.InlineDataWithTarget(cat.Program(), "/app/cat.wasm"),
					},
					Engine: wasmmodels.NewWasmEngineBuilder("/app/cat.wasm").
						WithEntrypoint("_start").
						WithParameters(path.Join(urlmount, urlfile), path.Join(localMount, localFile)).
						MustBuild(),
					Publisher: publisher_local.NewSpecConfig(),
				},
			},
		},
		ResultsChecker: scenario.FileEquals(downloader.DownloadFilenameStdout, URLContent+localContent),
		JobCheckers:    scenario.WaitUntilSuccessful(1),
	}

	s.RunScenario(testScenario)
}
