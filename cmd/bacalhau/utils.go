package bacalhau

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

const (
	JSONFormat                         string = "json"
	YAMLFormat                         string = "yaml"
	DefaultDockerRunWaitSeconds               = 600
	PrintoutCanceledButRunningNormally string = "printout canceled but running normally"
	// AutoDownloadFolderPerm is what permissions we give to a folder we create when downloading results
	AutoDownloadFolderPerm       = 0755
	HowFrequentlyToUpdateTicker  = 50 * time.Millisecond
	DefaultSpinnerFormatDuration = 30 * time.Millisecond
	DefaultTimeout               = 30 * time.Minute
)

var eventsWorthPrinting = map[model.JobStateType]eventStruct{
	model.JobStateNew:        {Message: "Creating job for submission", IsTerminal: false, PrintDownload: false, IsError: false},
	model.JobStateQueued:     {Message: "Job waiting to be scheduled", PrintDownload: true, IsTerminal: false, IsError: false},
	model.JobStateInProgress: {Message: "Job in progress", PrintDownload: true, IsTerminal: false, IsError: false},
	model.JobStateError:      {Message: "Error while executing the job", PrintDownload: false, IsTerminal: true, IsError: true},
	model.JobStateCancelled:  {Message: "Job canceled", PrintDownload: false, IsTerminal: true, IsError: false},
	model.JobStateCompleted:  {Message: "Job finished", PrintDownload: false, IsTerminal: true, IsError: false},
	model.JobStateCompletedPartially: {
		Message:       "Job partially complete",
		PrintDownload: false,
		IsTerminal:    true,
		IsError:       true,
	},
}

type eventStruct struct {
	Message       string
	IsTerminal    bool
	PrintDownload bool
	IsError       bool
}

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
	if len(id) < model.ShortIDLength {
		return id
	}
	return id[:model.ShortIDLength]
}

func GetAPIHostAndPort() string {
	return fmt.Sprintf("%s:%d", apiHost, apiPort)
}

func GetAPIClient() *publicapi.RequesterAPIClient {
	return publicapi.NewRequesterAPIClient(apiHost, apiPort)
}

// ensureValidVersion checks that the server version is the same or less than the client version
func ensureValidVersion(ctx context.Context, clientVersion, serverVersion *model.BuildVersionInfo) error {
	if clientVersion == nil {
		log.Ctx(ctx).Warn().Msg("Unable to parse nil client version, skipping version check")
		return nil
	}
	if clientVersion.GitVersion == version.DevelopmentGitVersion {
		log.Ctx(ctx).Debug().Msg("Development client version, skipping version check")
		return nil
	}
	if serverVersion == nil {
		log.Ctx(ctx).Warn().Msg("Unable to parse nil server version, skipping version check")
		return nil
	}
	if serverVersion.GitVersion == version.DevelopmentGitVersion {
		log.Ctx(ctx).Debug().Msg("Development server version, skipping version check")
		return nil
	}
	c, err := semver.NewVersion(clientVersion.GitVersion)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Unable to parse client version, skipping version check")
		return nil
	}
	s, err := semver.NewVersion(serverVersion.GitVersion)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("Unable to parse server version, skipping version check")
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

func NewIPFSDownloadFlags(settings *model.DownloaderSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("IPFS Download flags", pflag.ContinueOnError)
	flags.BoolVar(&settings.Raw, "raw",
		settings.Raw, "Download raw result CIDs instead of merging multiple CIDs into a single result")
	flags.DurationVar(&settings.Timeout, "download-timeout-secs",
		settings.Timeout, "Timeout duration for IPFS downloads.")
	flags.StringVar(&settings.OutputDir, "output-dir",
		settings.OutputDir, "Directory to write the output to.")
	flags.StringVar(&settings.IPFSSwarmAddrs, "ipfs-swarm-addrs",
		settings.IPFSSwarmAddrs, "Comma-separated list of IPFS nodes to connect to.")
	return flags
}

func getDefaultJobFolder(jobID string) string {
	return fmt.Sprintf("job-%s", system.GetShortID(jobID))
}

// if the user does not supply a value for "download results to here"
// then we default to making a folder in the current directory
func ensureDefaultDownloadLocation(jobID string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	downloadDir := filepath.Join(cwd, getDefaultJobFolder(jobID))
	err = os.MkdirAll(downloadDir, AutoDownloadFolderPerm)
	if err != nil {
		return "", err
	}
	return downloadDir, nil
}

func processDownloadSettings(settings model.DownloaderSettings, jobID string) (model.DownloaderSettings, error) {
	if settings.OutputDir == "" {
		dir, err := ensureDefaultDownloadLocation(jobID)
		if err != nil {
			return settings, err
		}
		settings.OutputDir = dir
	}
	return settings, nil
}

type RunTimeSettings struct {
	AutoDownloadResults   bool // Automatically download the results after finishing
	IPFSGetTimeOut        int  // Timeout for IPFS in seconds
	IsLocal               bool // Job should be executed locally
	WaitForJobToFinish    bool // Wait for the job to finish before returning
	WaitForJobTimeoutSecs int  // Timeout for waiting for the job to finish
	PrintJobIDOnly        bool // Only print the Job ID as output
	PrintNodeDetails      bool // Print the node details as output
	Follow                bool // Follow along with the output of the job
}

func NewRunTimeSettings() *RunTimeSettings {
	return &RunTimeSettings{
		AutoDownloadResults:   false,
		WaitForJobToFinish:    true,
		WaitForJobTimeoutSecs: DefaultDockerRunWaitSeconds,
		IPFSGetTimeOut:        10,
		IsLocal:               false,
		PrintJobIDOnly:        false,
		PrintNodeDetails:      false,
		Follow:                false,
	}
}

func NewRunTimeSettingsFlags(settings *RunTimeSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("Runtime settings", pflag.ContinueOnError)
	flags.IntVarP(&settings.IPFSGetTimeOut, "gettimeout", "g", settings.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`)
	flags.BoolVar(&settings.IsLocal, "local", settings.IsLocal,
		`Run the job locally. Docker is required`)
	flags.BoolVar(&settings.WaitForJobToFinish, "wait", settings.WaitForJobToFinish,
		`Wait for the job to finish.`)
	flags.IntVar(&settings.WaitForJobTimeoutSecs, "wait-timeout-secs", settings.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`)
	flags.BoolVar(&settings.PrintJobIDOnly, "id-only", settings.PrintJobIDOnly,
		`Print out only the Job ID on successful submission.`)
	flags.BoolVar(&settings.PrintNodeDetails, "node-details", settings.PrintNodeDetails,
		`Print out details of all nodes (overridden by --id-only).`)
	flags.BoolVar(&settings.AutoDownloadResults, "download", settings.AutoDownloadResults,
		`Should we download the results once the job is complete?`)
	flags.BoolVarP(&settings.Follow, "follow", "f", settings.Follow,
		`When specified will follow the output from the job as it runs`)

	return flags
}

func getCommandLineExecutable() string {
	return os.Args[0]
}

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	cm *system.CleanupManager,
	cmd *cobra.Command,
	j *model.Job,
	settings *ExecutionSettings,
) error {

	client := GetAPIClient()
	if settings.Runtime.IsLocal {
		stack, errLocalDevStack := devstack.NewDevStackForRunLocal(ctx, cm, 1, capacity.ConvertGPUString(j.Spec.Resources.GPU))
		if errLocalDevStack != nil {
			return errLocalDevStack
		}

		apiServer := stack.Nodes[0].APIServer
		client = publicapi.NewRequesterAPIClient(apiServer.Address, apiServer.Port)
	}

	err := job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return err
	}

	j, err = submitJob(ctx, client, j)
	if err != nil {
		return err
	}

	return printJobExecution(ctx, cmd, cm, client, j, settings)
}

func submitJob(ctx context.Context,
	client *publicapi.RequesterAPIClient,
	j *model.Job,
) (*model.Job, error) {
	j, err := client.Submit(ctx, j)
	if err != nil {
		return &model.Job{}, errors.Wrap(err, "failed to submit job")
	}
	return j, err
}

func ReadFromStdinIfAvailable(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) == 0 {
		r := bufio.NewReader(cmd.InOrStdin())
		reader := bufio.NewReader(r)

		// buffered channel of dataStream
		dataStream := make(chan []byte, 1)

		// Run scanner.Bytes() function in it's own goroutine and pass back it's
		// response into dataStream channel.
		go func() {
			for {
				res, err := reader.ReadBytes("\n"[0])
				if err != nil {
					break
				}
				dataStream <- res
			}
			close(dataStream)
		}()

		// Listen on dataStream channel AND a timeout channel - which ever happens first.
		var err error
		var bytesResult bytes.Buffer
		timedOut := false
		select {
		case res := <-dataStream:
			_, err = bytesResult.Write(res)
			if err != nil {
				return nil, err
			}
		case <-time.After(time.Duration(10) * time.Millisecond): //nolint:gomnd // 10ms timeout
			timedOut = true
		}

		if timedOut {
			cmd.Println("No input provided, waiting ... (Ctrl+D to complete)")
		}

		for read := range dataStream {
			_, err = bytesResult.Write(read)
		}

		return bytesResult.Bytes(), err
	}
	return nil, fmt.Errorf("should not be possible, args should be empty")
}

// WaitForJobAndPrintResultsToUser uses events to decide what to output to the terminal
// using a spinner to show long-running tasks. When the job is complete (or the user
// triggers SIGINT) then the function will complete and stop outputting to the terminal.
//
//nolint:gocyclo,funlen
func WaitForJobAndPrintResultsToUser(ctx context.Context, cmd *cobra.Command, j *model.Job, quiet bool) error {
	if j == nil || j.Metadata.ID == "" {
		return errors.New("No job returned from the server.")
	}

	getMoreInfoString := fmt.Sprintf(`
To get more information at any time, run:
   bacalhau describe %s`, j.Metadata.ID)

	cancelString := fmt.Sprintf(`
To cancel the job, run:
   bacalhau cancel %s`, j.Metadata.ID)

	if !quiet {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", j.Metadata.ID)
		cmd.Printf("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
	}

	// Create a map of job state types -> boolean, this is a record of what has been printed
	// so far.
	printedEventsTracker := make(map[model.JobStateType]bool)
	for _, jobEventType := range model.JobStateTypes() {
		printedEventsTracker[jobEventType] = false
	}

	// Inject "Job Initiated Event" to start - should we do this on the server?
	// TODO: #1068 Should jobs auto add a "start event" on the client at creation?
	// Faking an initial time (sometimes it happens too fast to see)
	// fullLineMessage.TimerString = spinnerFmtDuration(DefaultSpinnerFormatDuration)
	startMessage := "Communicating with the network"

	// Decide where the spinner should write it's output, by default we want stdout
	// but we can also disable output here if we have been asked to be quiet.
	writer := cmd.OutOrStdout()
	if quiet {
		writer = io.Discard
	}

	widestString := len(startMessage)
	for _, v := range eventsWorthPrinting {
		widestString = system.Max(widestString, len(v.Message))
	}

	spinner, err := NewSpinner(ctx, writer, widestString, false)
	if err != nil {
		Fatal(cmd, err.Error(), 1)
	}
	spinner.Run()
	spinner.NextStep(startMessage)

	// Capture Ctrl+C if the user wants to finish early the job
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, ShutdownSignals...)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	cmdShuttingDown := false
	var returnError error = nil

	// goroutine for handling SIGINT from the signal channel, or context
	// completion messages.
	go func() {
		for {
			select {
			case s := <-signalChan: // first signal, cancel context
				log.Ctx(ctx).Debug().Msgf("Captured %v. Exiting...", s)
				if s == os.Interrupt {
					cmdShuttingDown = true
					spinner.Done(StopCancel)

					if !quiet {
						cmd.Println("\n\n\rPrintout canceled (the job is still running).")
						cmd.Println(getMoreInfoString)
						cmd.Println(cancelString)
					}
					returnError = fmt.Errorf(PrintoutCanceledButRunningNormally)
				} else {
					cmd.Println("Unexpected signal received. Exiting.")
				}
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	var lastSeenTimestamp int64 = 0
	var lastEventState model.JobStateType
	for !cmdShuttingDown {
		// Get the job level history events that happened since the last one we saw
		jobEvents, err := GetAPIClient().GetEvents(ctx, j.Metadata.ID, publicapi.EventFilterOptions{
			Since:                 lastSeenTimestamp,
			ExcludeExecutionLevel: true,
		})
		if err != nil {
			if _, ok := err.(*bacerrors.ContextCanceledError); ok {
				// We're done, the user canceled the job
				cmdShuttingDown = true
				continue
			} else {
				return errors.Wrap(err, "Error getting job events")
			}
		}

		// Iterate through the events, looking for ones we have not yet processed
		for _, event := range jobEvents {
			if event.Time.Unix() > lastSeenTimestamp {
				lastSeenTimestamp = event.Time.Unix()
			}

			if event.Type != model.JobHistoryTypeJobLevel {
				continue
			}

			if !quiet {
				jet := event.JobState.New // Get the type of the new state
				wasPrinted := printedEventsTracker[jet]

				// If it hasn't been printed yet, we'll print this event.
				// We'll also skip lines where there's no message to print.
				if !wasPrinted && eventsWorthPrinting[jet].Message != "" {
					printedEventsTracker[jet] = true

					// We shouldn't do anything with execution errors because there could
					// be retries following, so for now we will
					if !eventsWorthPrinting[jet].IsError && !eventsWorthPrinting[jet].IsTerminal {
						spinner.NextStep(eventsWorthPrinting[jet].Message)
					}

					if event.JobState.New == model.JobStateQueued {
						spinner.msgMutex.Lock()
						spinner.msg.Waiting = true
						spinner.msg.Detail = event.Comment
						spinner.msgMutex.Unlock()
					}
				} else if wasPrinted && lastEventState == event.JobState.New {
					// Printing the same again but with new information, so we
					// can just update the existing message.
					spinner.msgMutex.Lock()
					spinner.msg.Detail = event.Comment
					spinner.msgMutex.Unlock()
				}
			}

			lastEventState = event.JobState.New

			if event.JobState.New.IsTerminal() {
				if event.JobState.New != model.JobStateCompleted {
					returnError = errors.New(event.Comment)
					spinner.Done(StopFailed)
				} else {
					spinner.Done(StopSuccess)
				}
				cmdShuttingDown = true
				break
			}
		}

		// Have we been cancel(l)ed?
		if condition := ctx.Err(); condition != nil {
			break
		}

		time.Sleep(time.Duration(500) * time.Millisecond) //nolint:gomnd // 500ms sleep
	}

	return returnError
}

func FatalErrorHandler(cmd *cobra.Command, msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		cmd.Print(msg)
	}
	os.Exit(code)
}

// FakeFatalErrorHandler captures the error for testing, responsibility of the test to handle the exit (if any)
// NOTE: If your test is not idempotent, you can cause side effects
// (the underlying function will continue to run)
// Returned as text JSON to wherever RootCmd is printing.
func FakeFatalErrorHandler(cmd *cobra.Command, msg string, code int) {
	c := model.TestFatalErrorHandlerContents{Message: msg, Code: code}
	b, _ := model.JSONMarshalWithMax(c)
	cmd.Println(string(b))
}

// applyPorcelainLogLevel sets the log level of loggers running on user-facing
// "porcelain" commands to be zerolog.FatalLevel to reduce noise shown to users.
func applyPorcelainLogLevel(cmd *cobra.Command, _ []string) {
	if _, err := zerolog.ParseLevel(os.Getenv("LOG_LEVEL")); err != nil {
		return
	}

	ctx := cmd.Context()
	ctx = log.Ctx(ctx).Level(zerolog.FatalLevel).WithContext(ctx)
	cmd.SetContext(ctx)
}

// DockerImageContainsTag checks if the image contains a tag or a digest
func DockerImageContainsTag(image string) bool {
	if strings.Contains(image, ":") {
		return true
	}
	if strings.Contains(image, "@") {
		return true
	}
	return false
}

type ExecutionSettings struct {
	Runtime  RunTimeSettings
	Download model.DownloaderSettings
}

func ExecuteDockerJob(ctx context.Context, cm *system.CleanupManager, cmd *cobra.Command, j *model.DockerJob, settings *ExecutionSettings) error {
	client := GetAPIClient()

	if settings.Runtime.IsLocal {
		stack, errLocalDevStack := devstack.NewDevStackForRunLocal(ctx, cm, 1, capacity.ConvertGPUString(j.ResourceConfig.GPU))
		if errLocalDevStack != nil {
			return errLocalDevStack
		}
		apiServer := stack.Nodes[0].APIServer
		client = publicapi.NewRequesterAPIClient(apiServer.Address, apiServer.Port)
	}

	// NOTE: job is verified when it constructed, we don't need to validate again.
	jobSpec, err := client.SubmitDockerJob(ctx, j)
	if err != nil {
		return fmt.Errorf("submitting docker job: %w", err)
	}

	return printJobExecution(ctx, cmd, cm, client, jobSpec, settings)
}

func ExecuteWasmJob(ctx context.Context, cm *system.CleanupManager, cmd *cobra.Command, j *model.WasmJob, settings *ExecutionSettings) error {
	client := GetAPIClient()

	if settings.Runtime.IsLocal {
		stack, errLocalDevStack := devstack.NewDevStackForRunLocal(ctx, cm, 1, capacity.ConvertGPUString(j.ResourceConfig.GPU))
		if errLocalDevStack != nil {
			return errLocalDevStack
		}
		apiServer := stack.Nodes[0].APIServer
		client = publicapi.NewRequesterAPIClient(apiServer.Address, apiServer.Port)
	}

	jobSpec, err := client.SubmitWasmJob(ctx, j)
	if err != nil {
		return fmt.Errorf("submitting wasm job: %w", err)
	}

	return printJobExecution(ctx, cmd, cm, client, jobSpec, settings)
}

func printJobExecution(ctx context.Context, cmd *cobra.Command, cm *system.CleanupManager, client *publicapi.RequesterAPIClient, j *model.Job, settings *ExecutionSettings) error {
	// if we are in --wait=false - print the id then exit
	// because all code after this point is related to
	// "wait for the job to finish" (via WaitForJobAndPrintResultsToUser)
	if !settings.Runtime.WaitForJobToFinish {
		cmd.Print(j.Metadata.ID + "\n")
		return nil
	}

	// if we are in --id-only mode - print the id
	if settings.Runtime.PrintJobIDOnly {
		cmd.Print(j.Metadata.ID + "\n")
	}

	if settings.Runtime.Follow {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", j.Metadata.ID)
		cmd.Printf("Waiting for logs... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")

		// Wait until the job has actually been accepted and started, otherwise this will fail waiting for
		// the execution to appear.
		for i := 0; i < 10; i++ {
			jobState, stateErr := client.GetJobState(ctx, j.ID())
			if stateErr != nil {
				Fatal(cmd, fmt.Sprintf("failed waiting for execution to start: %s", stateErr), 1)
			}

			executionID := ""
			for _, execution := range jobState.Executions {
				if execution.State.IsActive() {
					executionID = execution.ComputeReference
				}
			}

			if executionID != "" {
				break
			}
			time.Sleep(time.Duration(1) * time.Second)
		}

		logOptions := LogCommandOptions{WithHistory: true, Follow: true}
		return logs(cmd, []string{j.Metadata.ID}, logOptions)
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := settings.Runtime.PrintJobIDOnly

	jobErr := WaitForJobAndPrintResultsToUser(ctx, cmd, j, quiet)
	if jobErr != nil {
		if jobErr.Error() == PrintoutCanceledButRunningNormally {
			Fatal(cmd, "", 0)
		} else {
			cmd.PrintErrf("\nError submitting job: %s", jobErr)
		}
	}

	jobReturn, found, err := client.Get(ctx, j.Metadata.ID)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error getting job: %s", err), 1)
	}
	if !found {
		Fatal(cmd, fmt.Sprintf("Weird. Just ran the job, but we couldn't find it. Should be impossible. ID: %s", j.Metadata.ID), 1)
	}

	js, err := client.GetJobState(ctx, jobReturn.Job.Metadata.ID)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error getting job state: %s", err), 1)
	}

	if settings.Runtime.PrintNodeDetails || jobErr != nil {
		cmd.Println("\nJob Results By Node:")
		for message, nodes := range summariseExecutions(js) {
			cmd.Printf("• Node %s: ", strings.Join(nodes, ", "))
			if strings.ContainsRune(message, '\n') {
				cmd.Printf("\n\t%s\n", strings.Join(strings.Split(message, "\n"), "\n\t"))
			} else {
				cmd.Println(message)
			}
		}
	}

	hasResults := slices.ContainsFunc(js.Executions, func(e model.ExecutionState) bool { return e.RunOutput != nil })
	if !quiet && hasResults {
		cmd.Printf("\nTo download the results, execute:\n\t%s get %s\n", getCommandLineExecutable(), j.ID())
	}

	if !quiet {
		cmd.Printf("\nTo get more details about the run, execute:\n\t%s describe %s\n", getCommandLineExecutable(), j.ID())
	}

	if hasResults && settings.Runtime.AutoDownloadResults {
		err = downloadResultsHandler(
			ctx,
			cm,
			cmd,
			j.Metadata.ID,
			settings.Download,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// Groups the executions in the job state, returning a map of printable messages
// to node(s) that generated that message.
func summariseExecutions(state model.JobState) map[string][]string {
	results := make(map[string][]string, len(state.Executions))
	for _, execution := range state.Executions {
		var message string
		if execution.RunOutput != nil {
			if execution.RunOutput.ErrorMsg != "" {
				message = execution.RunOutput.ErrorMsg
			} else if execution.RunOutput.ExitCode > 0 {
				message = execution.RunOutput.STDERR
			} else {
				message = execution.RunOutput.STDOUT
			}
		} else if execution.State.IsDiscarded() {
			message = execution.Status
		}

		if message != "" {
			results[message] = append(results[message], system.GetShortID(execution.NodeID))
		}
	}
	return results
}

func downloadResultsHandler(
	ctx context.Context,
	cm *system.CleanupManager,
	cmd *cobra.Command,
	jobID string,
	downloadSettings model.DownloaderSettings,
) error {
	cmd.PrintErrf("Fetching results of job '%s'...\n", jobID)
	j, _, err := GetAPIClient().Get(ctx, jobID)

	if err != nil {
		if _, ok := err.(*bacerrors.JobNotFound); ok {
			return err
		} else {
			Fatal(cmd, fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", jobID, err), 1)
		}
	}

	results, err := GetAPIClient().GetResults(ctx, j.Job.Metadata.ID)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	processedDownloadSettings, err := processDownloadSettings(downloadSettings, j.Job.Metadata.ID)
	if err != nil {
		return err
	}

	downloaderProvider := util.NewStandardDownloaders(cm, &processedDownloadSettings)
	if err != nil {
		return err
	}

	// check if we don't support downloading the results
	for _, result := range results {
		if !downloaderProvider.Has(ctx, result.Data.Schema) {
			cmd.PrintErrln(
				"No supported downloader found for the published results. You will have to download the results differently.")
			b, err := json.MarshalIndent(results, "", "    ")
			if err != nil {
				return err
			}
			cmd.PrintErrln(string(b))
			return nil
		}
	}

	err = downloader.DownloadResults(
		ctx,
		results,
		downloaderProvider,
		&processedDownloadSettings,
	)

	if err != nil {
		return err
	}

	cmd.Printf("Results for job '%s' have been written to...\n", jobID)
	cmd.Printf("%s\n", processedDownloadSettings.OutputDir)

	return nil
}
