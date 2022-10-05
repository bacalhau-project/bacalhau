package bacalhau

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/userstrings"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	JSONFormat                  string = "json"
	YAMLFormat                  string = "yaml"
	DefaultDockerRunWaitSeconds        = 600
)

func shortenTime(outputWide bool, t time.Time) string { //nolint:unused // Useful function, holding here
	if outputWide {
		return t.Format("06-01-02-15:04:05")
	}

	return t.Format("15:04:05")
}

var DefaultShortenStringLength = 20

func shortenString(outputWide bool, st string) string {
	if outputWide {
		return st
	}

	if len(st) < DefaultShortenStringLength {
		return st
	}

	return st[:20] + "..."
}

func shortID(outputWide bool, id string) string {
	if outputWide {
		return id
	}
	return id[:8]
}

func GetAPIClient() *publicapi.APIClient {
	return publicapi.NewAPIClient(fmt.Sprintf("http://%s:%d", apiHost, apiPort))
}

// ensureValidVersion checks that the server version is the same or less than the client version
func ensureValidVersion(ctx context.Context, clientVersion, serverVersion *model.BuildVersionInfo) error {
	if clientVersion == nil {
		log.Warn().Msg("Unable to parse nil client version, skipping version check")
		return nil
	}
	if clientVersion.GitVersion == "v0.0.0-xxxxxxx" {
		log.Debug().Msg("Development client version, skipping version check")
		return nil
	}
	if serverVersion == nil {
		log.Warn().Msg("Unable to parse nil server version, skipping version check")
		return nil
	}
	if serverVersion.GitVersion == "v0.0.0-xxxxxxx" {
		log.Debug().Msg("Development server version, skipping version check")
		return nil
	}
	c, err := semver.NewVersion(clientVersion.GitVersion)
	if err != nil {
		log.Warn().Err(err).Msg("Unable to parse client version, skipping version check")
		return nil
	}
	s, err := semver.NewVersion(serverVersion.GitVersion)
	if err != nil {
		log.Warn().Err(err).Msg("Unable to parse server version, skipping version check")
		return nil
	}
	if s.GreaterThan(c) {
		return fmt.Errorf(`the server version %s is newer than client version %s, please upgrade your client with the following command:
curl -sL https://get.bacalhau.org/install.sh | bash`,
			serverVersion.GitVersion,
			clientVersion.GitVersion,
		)
	}
	if c.GreaterThan(s) {
		return fmt.Errorf(
			"client version %s is newer than server version %s, please ask your network administrator to update Bacalhau",
			clientVersion.GitVersion,
			serverVersion.GitVersion,
		)
	}
	return nil
}

func ExecuteTestCobraCommand(_ *testing.T, root *cobra.Command, args ...string) (
	c *cobra.Command, output string, err error,
) { //nolint:unparam // use of t is valuable here
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{})
	root.SetArgs(args)

	// Need to check if we're running in debug mode for VSCode
	// Empty them if they exist
	if (len(os.Args) > 2) && (os.Args[1] == "-test.run") {
		os.Args[1] = ""
		os.Args[2] = ""
	}

	log.Trace().Msgf("Command to execute: %v", root.CalledAs())

	c, err = root.ExecuteC()
	return c, buf.String(), err
}

// this function captures the output of all functions running in it between capture() and done()
// example:
// 	done := capture()
//	fmt.Println("hello")
//	s, _ := done()
// after trimming str := strings.TrimSpace(s) it will return "hello"
// so if we want to compare the output in the console with a expected output like "hello" we could do that
// this is mainly used in testing --local
// go playground link https://go.dev/play/p/cuGIaIorWfD

//nolint:unused
func capture() func() (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	done := make(chan error, 1)

	save := os.Stdout
	os.Stdout = w

	var buf strings.Builder

	go func() {
		_, err := io.Copy(&buf, r)
		r.Close()
		done <- err
	}()

	return func() (string, error) {
		os.Stdout = save
		w.Close()
		err := <-done
		return buf.String(), err
	}
}

func setupDownloadFlags(cmd *cobra.Command, settings *ipfs.IPFSDownloadSettings) {
	cmd.Flags().IntVar(&settings.TimeoutSecs, "download-timeout-secs",
		settings.TimeoutSecs, "Timeout duration for IPFS downloads.")
	cmd.Flags().StringVar(&settings.OutputDir, "output-dir",
		settings.OutputDir, "Directory to write the output to.")
	cmd.Flags().StringVar(&settings.IPFSSwarmAddrs, "ipfs-swarm-addrs",
		settings.IPFSSwarmAddrs, "Comma-separated list of IPFS nodes to connect to.")
}

type RunTimeSettings struct {
	WaitForJobToFinish               bool // Wait for the job to execute before exiting
	WaitForJobToFinishAndPrintOutput bool // Wait for the job to execute, and print the results before exiting
	WaitForJobTimeoutSecs            int  // Job time out in seconds
	IPFSGetTimeOut                   int  // Timeout for IPFS in seconds
	IsLocal                          bool // Job should be executed locally

}

func NewRunTimeSettings() *RunTimeSettings {
	return &RunTimeSettings{
		WaitForJobToFinish:               false,
		WaitForJobToFinishAndPrintOutput: false,
		WaitForJobTimeoutSecs:            DefaultDockerRunWaitSeconds,
		IPFSGetTimeOut:                   10,
		IsLocal:                          false,
	}
}

func setupRunTimeFlags(cmd *cobra.Command, settings *RunTimeSettings) {
	cmd.PersistentFlags().BoolVar(
		&settings.WaitForJobToFinish, "wait", settings.WaitForJobToFinish,
		`Wait for the job to finish.`,
	)

	cmd.PersistentFlags().IntVarP(
		&settings.IPFSGetTimeOut, "gettimeout", "g", settings.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`,
	)

	cmd.PersistentFlags().BoolVar(
		&settings.IsLocal, "local", settings.IsLocal,
		`Run the job locally. Docker is required`,
	)

	cmd.PersistentFlags().BoolVar(
		&settings.WaitForJobToFinishAndPrintOutput, "download", settings.WaitForJobToFinishAndPrintOutput,
		`Download the results and print stdout once the job has completed (implies --wait).`,
	)

	cmd.PersistentFlags().IntVar(
		&settings.WaitForJobTimeoutSecs, "wait-timeout-secs", settings.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`,
	)
}

func ExecuteJob(ctx context.Context,
	cm *system.CleanupManager,
	cmd *cobra.Command,
	j *model.Job,
	runtimeSettings RunTimeSettings,
	downloadSettings ipfs.IPFSDownloadSettings,
) error {
	var apiClient *publicapi.APIClient
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/utils.ExecuteJob")
	defer span.End()

	if runtimeSettings.IsLocal {
		stack, errLocalDevStack := devstack.NewDevStackForRunLocal(ctx, cm, 1, j.Spec.Resources.GPU)
		if errLocalDevStack != nil {
			return errLocalDevStack
		}

		apiURI := stack.Nodes[0].APIServer.GetURI()
		apiClient = publicapi.NewAPIClient(apiURI)
	} else {
		apiClient = GetAPIClient()
	}

	err := job.VerifyJob(j)
	if err != nil {
		log.Err(err).Msg("Job failed to validate.")
		return err
	}

	j, err = submitJob(ctx, apiClient, j)
	if err != nil {
		return err
	}

	err = PrintReturnedJobIDToUser(j)
	if err != nil {
		Fatal(fmt.Sprintf("Error submitting job: %s", err), 1)
	}

	if runtimeSettings.WaitForJobToFinish || runtimeSettings.WaitForJobToFinishAndPrintOutput {
		// We have a jobID now, add it to the context baggage
		ctx = system.AddJobIDToBaggage(ctx, j.ID)
		system.AddJobIDFromBaggageToSpan(ctx, span)

		resolver := apiClient.GetJobStateResolver()
		resolver.SetWaitTime(ODR.RunTimeSettings.WaitForJobTimeoutSecs, time.Second*1)
		err = resolver.WaitUntilComplete(ctx, j.ID)
		if err != nil {
			return err
		}

		err := waitForJobToFinish(ctx, apiClient, j, runtimeSettings)
		if err != nil {
			return err
		}
		if runtimeSettings.WaitForJobToFinishAndPrintOutput {
			results, err := getResults(ctx, apiClient, j)
			if err != nil {
				return errors.Wrap(err, "cmd/bacalhau/utils/ExecuteJob: error getting results")
			}

			if len(results) == 0 {
				return fmt.Errorf("no results found")
			}

			err = downloadResults(ctx, cmd, cm, j, results, downloadSettings)
			if err != nil {
				return errors.Wrap(err, "cmd/bacalhau/utils/ExecuteJob: error downloading results")
			}
		}
	}
	return nil
}

func waitForJobToFinish(ctx context.Context,
	apiClient *publicapi.APIClient,
	j *model.Job,
	runtimeSettings RunTimeSettings) error {
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/utils.waitForJobToFinish")
	defer span.End()

	resolver := apiClient.GetJobStateResolver()
	resolver.SetWaitTime(runtimeSettings.WaitForJobTimeoutSecs, time.Second*1)
	err := resolver.WaitUntilComplete(ctx, j.ID)
	if err != nil {
		return err
	}

	return nil
}

func submitJob(ctx context.Context,
	apiClient *publicapi.APIClient,
	j *model.Job) (*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/utils.submitJob")
	defer span.End()

	j, err := apiClient.Submit(ctx, j, nil)
	if err != nil {
		return &model.Job{}, errors.Wrap(err, "failed to submit job")
	}
	return j, err
}

func getResults(ctx context.Context, apiClient *publicapi.APIClient, j *model.Job) ([]model.StorageSpec, error) {
	ctx, span := system.GetTracer().Start(ctx, "getresults")
	defer span.End()

	results, err := apiClient.GetResults(ctx, j.ID)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found")
	}
	return results, nil
}

func downloadResults(ctx context.Context,
	cmd *cobra.Command,
	cm *system.CleanupManager,
	j *model.Job,
	results []model.StorageSpec,
	downloadSettings ipfs.IPFSDownloadSettings) error {
	ctx, span := system.GetTracer().Start(ctx, "downloadresults")
	defer span.End()

	err := ipfs.DownloadJob(
		ctx,
		cm,
		j,
		results,
		downloadSettings,
	)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(filepath.Join(downloadSettings.OutputDir, "stdout"))
	if err != nil {
		return err
	}
	cmd.Println()
	cmd.Println(string(body))

	return nil
}

func ReadFromStdinIfAvailable(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) == 0 {
		byteResult, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, errors.Wrap(err, "Error reading from stdin")
		}
		if byteResult == nil {
			cmd.Println(userstrings.NoFilenameProvidedErrorString)
			return nil, errors.New(userstrings.NoStdInProvidedErrorString)
		}
		return byteResult, nil
	}
	return nil, errors.New(userstrings.NoStdInProvidedErrorString)
}

func PrintReturnedJobIDToUser(j *model.Job) error {
	if j == nil || j.ID == "" {
		return errors.New("No job returned from the server.")
	}

	// so you can run job_id=$(bacalhau docker run ...)
	// DO NOT REGRESS THIS BEHAVIOR, THE EXAMPLES DEPEND ON IT
	RootCmd.Printf("%s\n", j.ID)

	// so the user knows they can use describe
	fmt.Fprintln(system.Stderr, "\nTo get the status of the job, run:")
	fmt.Fprintf(system.Stderr, "  bacalhau describe %s\n", j.ID)
	return nil
}

func FatalErrorHandler(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		RootCmd.Print(msg)
	}
	os.Exit(code)
}

// Captures for testing, responsibility of the test to handle the exit (if any)
// NOTE: If your test is not idempotent, you can cause side effects
// (the underlying function will continue to run)
// Returned as text JSON to wherever RootCmd is printing.
func FakeFatalErrorHandler(msg string, code int) {
	c := model.TestFatalErrorHandlerContents{Message: msg, Code: code}
	b, _ := json.Marshal(c)
	RootCmd.Println(string(b))
}
