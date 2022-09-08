package bacalhau

// import (
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"strconv"
// 	"strings"
// 	"testing"

// 	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
// 	"github.com/filecoin-project/bacalhau/pkg/computenode"
// 	"github.com/filecoin-project/bacalhau/pkg/system"
// 	"github.com/filecoin-project/bacalhau/pkg/test/devstack"
// 	"github.com/spf13/cobra"
// 	"github.com/stretchr/testify/require"
// 	"github.com/stretchr/testify/suite"
// )

// // Define the suite, and absorb the built-in basic suite
// // functionality from testify - including a T() method which
// // returns the current testing context
// type ExamplesTestSuite struct {
// 	suite.Suite
// 	rootCmd *cobra.Command
// }

// // In order for 'go test' to run this suite, we need to create
// // a normal test function and pass our suite to suite.Run
// func TestExamplesSuite(t *testing.T) {
// 	suite.Run(t, new(ExamplesTestSuite))
// }

// // Before all suite
// func (suite *ExamplesTestSuite) SetupAllSuite() {
// }

// // Before each test
// func (suite *ExamplesTestSuite) SetupTest() {
// 	err :=system.InitConfigForTesting()
//  require.NoError(t, err)
// 	suite.rootCmd = bacalhau.RootCmd
// }

// func (suite *ExamplesTestSuite) TearDownTest() {
// }

// func (suite *ExamplesTestSuite) TearDownAllSuite() {

// }

// func (suite *ExamplesTestSuite) TestExampleIntegrationTests() {
// 	if os.Getenv("RUN_INTEGRATION_TESTS") != strconv.Itoa(1) {
// 		suite.T().Skip("Skipping long running integration tests")
// 	}

// 	tests := []struct {
// 		args           []string
// 		expectedStdout string
// 	}{
// 		{args: []string{"docker", "run",
// 			"python",
// 			"--wait",
// 			"--download",
// 			"-v", "QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.py",
// 			"--",
// 			"/bin/bash", "-c", "python hello.py"}, expectedStdout: "hello world"},
// 	}
// 	devstack, cm := devstack.SetupTest(suite.T(), 1, 0, computenode.ComputeNodeConfig{})
// 	defer cm.Cleanup()

// 	for _, tc := range tests {
// 		func() {
// 			*bacalhau.ODR = *bacalhau.NewDockerRunOptions()
// 			tc.args = append(tc.args,
// 				"--api-host", devstack.Nodes[0].APIServer.Host,
// 				"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port))

// 			dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPython-")
// 			defer func() {
// 				err := os.RemoveAll(dir)
// 				require.NoError(suite.T(), err)
// 			}()
// 			bacalhau.ODR.DockerRunDownloadFlags.OutputDir = dir

// 			done := bacalhau.CaptureOutput()
// 			_, _, err := bacalhau.ExecuteTestCobraCommand(suite.T(), suite.rootCmd, tc.args...)
// 			out, _ := done()

// 			require.NoError(suite.T(), err)
// 			trimmedStdout := strings.TrimSpace(out)
// 			fmt.Println(trimmedStdout)

// 			require.Equal(suite.T(), tc.expectedStdout, trimmedStdout, "Expected %s as output, but got %s", tc.expectedStdout, trimmedStdout)
// 		}()
// 	}
// }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitDuckDB() {
// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/duckdb/stdout")
// // 	expectedStdout := strings.TrimSpace(string(content))
// // 	args := []string{"docker", "run",
// // 		"davidgasquez/datadex:v0.2.0",
// // 		"--wait",
// // 		"--download",
// // 		"-w", "/inputs/",
// // 		"--",
// // 		"/bin/bash", "-c", "duckdb -s 'select 1'"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPandas-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// //nolint:lll // commented out
// // // bacalhau -v QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files -w /files  docker run amancevice/pandas -- /bin/bash -c 'python read_csv.py'
// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitPandas() {
// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/pandas/stdout")
// // 	expectedStdout := strings.TrimSpace(string(content))
// // 	args := []string{"docker", "run",
// // 		"amancevice/pandas",
// // 		"--wait",
// // 		"--download",
// // 		"-v", "QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files",
// // 		"-w", "/files",
// // 		"--",
// // 		"/bin/bash", "-c", "python read_csv.py"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPandas-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitPytorch() {
// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/pytorch/stdout")
// // 	expectedStdout := string(content)
// // 	args := []string{"docker", "run",
// // 		"pytorch/pytorch",
// // 		"--wait",
// // 		"--download",
// // 		"-v", "QmZWPdWyuWxiJAqPC2nQXTp7P9geYNN75ZmnhoCT2jqnoe:/train.py",
// // 		"--",
// // 		"/bin/bash", "-c", "cd /;python train.py"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalSklearn-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout[:6], trimmedStdout[:6], "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitR() {
// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/r/stdout")
// // 	expectedStdout := string(content)
// // 	args := []string{"docker", "run",
// // 		"jsace/r-prophet",
// // 		"--download",
// // 		"--wait",
// // 		"-v", "QmZiwZz7fXAvQANKYnt7ya838VPpj4agJt5EDvRYp3Deeo:/input",
// // 		"-o", "output:/output",
// // 		"--",
// // 		"/bin/bash", "-c", "Rscript Saturating-Forecasts.R  input/example_wp_log_R.csv output/output0.pdf output/output1.pdf"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalSklearn-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout[:3], trimmedStdout[:3], "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."

// // }
// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitSklearn() {
// // 	expectedStdout := "[1]"
// // 	args := []string{"docker", "run",
// // 		"bitnami/scikit-learn-intel",
// // 		"--wait",
// // 		"--download",
// // 		"-v", "QmQ43onwDwW1kPZ9A4GxVY7n68DjG846S9AzPDLRV5T94b:/train.py",
// // 		"--",
// // 		"/bin/bash", "-c", "cd /;python train.py"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPython-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitTensorflow() {
// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/tensorflow/stdout")
// // 	expectedStdout := string(content)
// // 	args := []string{"docker", "run",
// // 		"tensorflow/tensorflow",
// // 		"--wait",
// // 		"--download",
// // 		"-v", "QmcWjFB2bSEhRLr6vwN2MZyDNvZNSqBtHK1dMk2c2uu2Bg:/train.py",
// // 		"--",
// // 		"/bin/bash", "-c", "cd /;python train.py"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalSklearn-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout[:5], trimmedStdout[:5], "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."

// // }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitPython() {
// // 	devstack, cm := devstack.SetupTest(suite.T(), 1, 0, computenode.ComputeNodeConfig{})
// // 	defer cm.Cleanup()

// // 	expectedStdout := "hello world"
// // 	args := []string{"docker", "run",
// // 		"--api-host", devstack.Nodes[0].APIServer.Host,
// // 		"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port),
// // 		"python",
// // 		"--wait",
// // 		"--download",
// // 		"-v", "QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.py",
// // 		"--",
// // 		"/bin/bash", "-c", "python hello.py"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPython-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitDuckDB() {
// // 	devstack, cm := devstack.SetupTest(suite.T(), 1, 0, computenode.ComputeNodeConfig{})
// // 	defer cm.Cleanup()

// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/duckdb/stdout")
// // 	expectedStdout := strings.TrimSpace(string(content))
// // 	args := []string{"docker", "run",
// // 		"davidgasquez/datadex:v0.2.0",
// // 		"--api-host", devstack.Nodes[0].APIServer.Host,
// // 		"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port),
// // 		"--wait",
// // 		"--download",
// // 		"-w", "/inputs/",
// // 		"--",
// // 		"/bin/bash", "-c", "duckdb -s 'select 1'"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPandas-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// //nolint:lll // commented out
// // // bacalhau -v QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files -w /files  docker run amancevice/pandas -- /bin/bash -c 'python read_csv.py'
// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitPandas() {
// // 	devstack, cm := devstack.SetupTest(suite.T(), 1, 0, computenode.ComputeNodeConfig{})
// // 	defer cm.Cleanup()

// // 	content, _ := ioutil.ReadFile("../../testdata/integrationdata/pandas/stdout")
// // 	expectedStdout := strings.TrimSpace(string(content))
// // 	args := []string{"docker", "run",
// // 		"amancevice/pandas",
// // 		"--api-host", devstack.Nodes[0].APIServer.Host,
// // 		"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port),
// // 		"--wait",
// // 		"--download",
// // 		"-v", "QmfKJT13h5k1b23ja3ZCVg5nFL9oKz2bVXc8oXgtwiwhjz:/files",
// // 		"-w", "/files",
// // 		"--",
// // 		"/bin/bash", "-c", "python read_csv.py"}

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPandas-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)

// // 	runDownloadFlags.OutputDir = "."
// // }

// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitLocalPython() {
// // 	*ODR = *NewDockerRunOptions()
// // 	CID := "QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk"
// // 	args := []string{"docker", "run",
// // 		"--local",
// // 		"--wait",
// // 		"--download",
// // 		"-v", fmt.Sprintf("%s:/hello.py", CID),
// // 		"python",
// // 		"--",
// // 		"/bin/bash", "-c", "python hello.py"}
// // 	expectedStdout := "hello"

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalPython-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()

// // 	ODR.DockerRunDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)
// // 	re := regexp.MustCompile(`(?m)[a-zA-Z]+`)
// // 	trimmedStdout = re.FindString(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)
// // }

// //nolint:lll // commented out
// // //  bacalhau docker run  --wait --download -v QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk:/hello.R r-base -- /bin/bash -c 'Rscript hello.R'
// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitLocalR() {
// // 	CID := "QmQRVx3gXVLaRXywgwo8GCTQ63fHqWV88FiwEqCidmUGhk"
// // 	args := []string{"docker", "run",
// // 		"--download",
// // 		"--local",
// // 		"-v", fmt.Sprintf("%s:/hello.R", CID),
// // 		"r-base",
// // 		"--",
// // 		"/bin/bash", "-c", "Rscript hello.R"}
// // 	expectedStdout := "hello"

// // 	dir, _ := ioutil.TempDir("", "bacalhau-TestRun_GenericSubmitLocalR-")
// // 	defer func() {
// // 		err := os.RemoveAll(dir)
// // 		require.NoError(suite.T(), err)
// // 	}()
// // 	runDownloadFlags.OutputDir = dir

// // 	done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)
// // 	re := regexp.MustCompile(`(?m)[a-zA-Z]+`)
// // 	trimmedStdout = re.FindString(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)
// // }
// // func (suite *ExamplesTestSuite) TestRun_GenericSubmitLocalOutput() {
// // 	devstack, cm := devstack.SetupTest(suite.T(), 1, 0, computenode.ComputeNodeConfig{})
// // 	defer cm.Cleanup()

// // 	args := []string{"docker", "run",
// // 		"ubuntu",
// // 		"--api-host", devstack.Nodes[0].APIServer.Host,
// // 		"--api-port", fmt.Sprintf("%d", devstack.Nodes[0].APIServer.Port),
// // 		"--local",
// // 		"--wait",
// // 		"--download",
// // 		"-w", "/outputs",
// // 		"--",
// // 		"/bin/bash", "-c", "printf hello > hello.txt"}
// // 	expectedStdout := "hello"

// // 	// done := capture()
// // 	_, _, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
// // 	if err != nil {
// // 		fmt.Print(err)
// // 	}
// // 	// out, _ := done()

// // 	require.NoError(suite.T(), err)
// // 	content, _ := ioutil.ReadFile("volumes/outputs/hello.txt")
// // 	out := string(content)
// // 	trimmedStdout := strings.TrimSpace(string(out))
// // 	fmt.Println(trimmedStdout)

// // 	require.Equal(suite.T(), expectedStdout, trimmedStdout, "Expected %s as output, but got %s", expectedStdout, trimmedStdout)
// // }
