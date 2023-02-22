package bacalhau

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/theckman/yacspin"
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

var eventsWorthPrinting = map[model.ExecutionStateType]eventStruct{
	model.ExecutionStateNew: {Message: "Creating job for submission", IsTerminal: false, PrintDownload: true, IsError: false},

	// Job is on Requester
	model.ExecutionStateAskForBid: {Message: "Finding node(s) for the job", IsTerminal: false, PrintDownload: true, IsError: false},

	// Job is on ComputeNode
	model.ExecutionStateBidAccepted: {Message: "Running the job", IsTerminal: false, PrintDownload: true, IsError: false},

	// Need to add a carriage return to the end of the line, but only this one
	model.ExecutionStateFailed: {Message: "Error while executing the job.", IsTerminal: true, PrintDownload: false, IsError: true},

	// Job is on StorageNode
	model.ExecutionStateResultProposed: {Message: "Job finished, verifying results", IsTerminal: false, PrintDownload: true, IsError: false},
	model.ExecutionStateResultRejected: {Message: "Results failed verification.", IsTerminal: true, PrintDownload: false, IsError: false},
	model.ExecutionStateResultAccepted: {Message: "Results accepted, publishing", IsTerminal: false, PrintDownload: true, IsError: false},
	model.ExecutionStateCompleted:      {Message: "", IsTerminal: true, PrintDownload: true, IsError: false},

	// Job is canceled by the user
	model.ExecutionStateCanceled: {Message: "Job canceled by the user.", IsTerminal: true, PrintDownload: false, IsError: true},
}

// Struct for tracking what's been printedEvents
type printedEvents struct {
	order   int
	printed bool
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

func GetAPIClient() *publicapi.RequesterAPIClient {
	return publicapi.NewRequesterAPIClient(fmt.Sprintf("http://%s:%d", apiHost, apiPort))
}

// ensureValidVersion checks that the server version is the same or less than the client version
func ensureValidVersion(ctx context.Context, clientVersion, serverVersion *model.BuildVersionInfo) error {
	if clientVersion == nil {
		log.Ctx(ctx).Warn().Msg("Unable to parse nil client version, skipping version check")
		return nil
	}
	if clientVersion.GitVersion == "v0.0.0-xxxxxxx" {
		log.Ctx(ctx).Debug().Msg("Development client version, skipping version check")
		return nil
	}
	if serverVersion == nil {
		log.Ctx(ctx).Warn().Msg("Unable to parse nil server version, skipping version check")
		return nil
	}
	if serverVersion.GitVersion == "v0.0.0-xxxxxxx" {
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

func ExecuteTestCobraCommand(t *testing.T, args ...string) (c *cobra.Command, output string, err error) {
	return ExecuteTestCobraCommandWithStdin(t, nil, args...)
}

func ExecuteTestCobraCommandWithStdin(_ *testing.T, stdin io.Reader, args ...string) (
	c *cobra.Command, output string, err error,
) { //nolint:unparam // use of t is valuable here
	buf := new(bytes.Buffer)
	root := NewRootCmd()
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(stdin)
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

func NewIPFSDownloadFlags(settings *model.DownloaderSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("IPFS Download flags", pflag.ContinueOnError)
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
	runtimeSettings RunTimeSettings,
	downloadSettings model.DownloaderSettings,
) error {
	var apiClient *publicapi.RequesterAPIClient

	if runtimeSettings.IsLocal {
		stack, errLocalDevStack := devstack.NewDevStackForRunLocal(ctx, cm, 1, capacity.ConvertGPUString(j.Spec.Resources.GPU))
		if errLocalDevStack != nil {
			return errLocalDevStack
		}

		apiURI := stack.Nodes[0].APIServer.GetURI()
		apiClient = publicapi.NewRequesterAPIClient(apiURI)
	} else {
		apiClient = GetAPIClient()
	}

	err := job.VerifyJob(ctx, j)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Job failed to validate.")
		return err
	}

	j, err = submitJob(ctx, apiClient, j)
	if err != nil {
		return err
	}

	// if we are in --wait=false - print the id then exit
	// because all code after this point is related to
	// "wait for the job to finish" (via WaitForJobAndPrintResultsToUser)
	if !runtimeSettings.WaitForJobToFinish {
		cmd.Print(j.Metadata.ID + "\n")
		return nil
	}

	// if we are in --id-only mode - print the id
	if runtimeSettings.PrintJobIDOnly {
		cmd.Print(j.Metadata.ID + "\n")
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := runtimeSettings.PrintJobIDOnly

	err = WaitForJobAndPrintResultsToUser(ctx, cmd, j, quiet)
	if err != nil {
		if err.Error() == PrintoutCanceledButRunningNormally {
			Fatal(cmd, "", 0)
		} else {
			Fatal(cmd, fmt.Sprintf("Error submitting job: %s", err), 1)
		}
	}

	jobReturn, found, err := apiClient.Get(ctx, j.Metadata.ID)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error getting job: %s", err), 1)
	}
	if !found {
		Fatal(cmd, fmt.Sprintf("Weird. Just ran the job, but we couldn't find it. Should be impossible. ID: %s", j.Metadata.ID), 1)
	}

	js, err := apiClient.GetJobState(ctx, jobReturn.Job.Metadata.ID)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error getting job state: %s", err), 1)
	}

	printOut := "%s" // We only know this at the end, we'll fill it in there.
	resultsCID := ""
	indentOne := "  "
	indentTwo := strings.Repeat(indentOne, 2)
	if runtimeSettings.PrintNodeDetails {
		printOut += "\n"
		printOut += "Job Results By Node:\n"
		for j, s := range js.Shards {
			printOut += fmt.Sprintf(indentOne+"Shard %d:\n", j)
			printOut += fmt.Sprintf(indentTwo+"State: %s\n", s.State)
			printOut += fmt.Sprintf(indentTwo+"Status: %s\n", s.State)
			for _, n := range s.Executions {
				printOut += fmt.Sprintf("Node %s:\n", n.NodeID[:8])
				if n.RunOutput == nil {
					printOut += fmt.Sprintf(indentTwo + "No RunOutput for this shard\n")
				} else {
					printOut += fmt.Sprintf(indentTwo+"Container Exit Code: %d\n", n.RunOutput.ExitCode)
					resultsCID = n.PublishedResult.CID // They're all the same, doesn't matter if we assign it many times
					printResults := func(t string, s string, trunc bool) {
						truncatedString := ""
						if trunc {
							truncatedString = " (truncated: last 2000 characters)"
						}
						if s != "" {
							printOut += fmt.Sprintf(indentTwo+"%s%s:\n      %s\n", t, truncatedString, s)
						} else {
							printOut += fmt.Sprintf(indentTwo+"%s%s: <NONE>\n", t, truncatedString)
						}
					}
					printResults("Stdout", n.RunOutput.STDOUT, n.RunOutput.StdoutTruncated)
					printResults("Stderr", n.RunOutput.STDERR, n.RunOutput.StderrTruncated)
				}
			}
		}
	}

	printOut += fmt.Sprintf(`
To download the results, execute:
%s%s get %s

To get more details about the run, execute:
%s%s describe %s
`, indentOne,
		getCommandLineExecutable(),
		j.Metadata.ID,
		indentOne,
		getCommandLineExecutable(),
		j.Metadata.ID)

	// Have to do a final Sprintf so we can inject the resultsCID into the right place
	if resultsCID != "" {
		resultsCID = fmt.Sprintf("Results CID: %s\n", resultsCID)
	}
	if !quiet {
		cmd.Print(fmt.Sprintf(printOut, resultsCID))
	}

	if runtimeSettings.AutoDownloadResults {
		err = downloadResultsHandler(
			ctx,
			cm,
			cmd,
			j.Metadata.ID,
			downloadSettings,
		)
		if err != nil {
			return err
		}
	}
	return nil
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

	err = downloader.DownloadJob(
		ctx,
		j.Job.Spec.Outputs,
		results,
		downloaderProvider,
		&processedDownloadSettings,
	)

	if err != nil {
		return err
	}

	cmd.PrintErrf("Results for job '%s' have been written to...\n", jobID)
	cmd.Printf("%s\n", processedDownloadSettings.OutputDir)

	return nil
}

func submitJob(ctx context.Context,
	apiClient *publicapi.RequesterAPIClient,
	j *model.Job,
) (*model.Job, error) {
	j, err := apiClient.Submit(ctx, j)
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

// FullLineMessage has to be global so that multiple routines can access
type FullLineMessage struct {
	Message     string
	TimerString string
	StopString  string
	Width       int
}

var fullLineMessage FullLineMessage

func (f *FullLineMessage) String() string {
	return fmt.Sprintf("%s %s ",
		f.Message,
		f.StopString)
}

func (f *FullLineMessage) PrintDone() string {
	return fmt.Sprintf("%s%s%s %s",
		f.String(),
		// Need to add 10 to have everything line up.
		strings.Repeat(".", f.Width+10), //nolint:gomnd // extra spacing
		" done ✅ ",
		f.TimerString)
}

func (f *FullLineMessage) PrintError() string {
	return fmt.Sprintf("%s%s%s %s",
		f.String(),
		// Need to add 10 to have everything line up.
		strings.Repeat(".", f.Width+10), //nolint:gomnd // extra spacing
		" err  ❌ ",
		f.TimerString)
}

const spacerText = " ... "

var ticker *time.Ticker
var tickerDone = make(chan bool)

// WaitForJobAndPrintResultsToUser uses events to decide what to output to the terminal
// using a spinner to show long-running tasks. When the job is complete (or the user
// triggers SIGINT) then the function will complete and stop outputting to the terminal.
//
//nolint:gocyclo,funlen
func WaitForJobAndPrintResultsToUser(ctx context.Context, cmd *cobra.Command, j *model.Job, quiet bool) error {
	fullLineMessage = FullLineMessage{
		Message:     "",
		TimerString: "",
		StopString:  "",
		Width:       6,
	}

	if j == nil || j.Metadata.ID == "" {
		return errors.New("No job returned from the server.")
	}
	getMoreInfoString := fmt.Sprintf(`
To get more information at any time, run:
   bacalhau describe %s`, j.Metadata.ID)

	if !quiet {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", j.Metadata.ID)
		cmd.Printf("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
	}

	// Create a map of job state types to printed structs
	var printedEventsTracker sync.Map
	for _, jobEventType := range model.ExecutionStateTypes() {
		printedEventsTracker.Store(jobEventType, printedEvents{
			printed: false,
			order:   int(jobEventType),
		})
	}

	time.Sleep(1 * time.Second)

	jobEvents, err := GetAPIClient().GetEvents(ctx, j.Metadata.ID)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Failure retrieving job events '%s': %s\n", j.Metadata.ID, err), 1)
	}

	// Inject "Job Initiated Event" to start - should we do this on the server?
	// TODO: #1068 Should jobs auto add a "start event" on the client at creation?
	// Faking an initial time (sometimes it happens too fast to see)
	fullLineMessage.TimerString = spinnerFmtDuration(DefaultSpinnerFormatDuration)
	fullLineMessage.Message = formatMessage("Communicating with the network")

	// Create a spinner var that will span all printouts
	spin, err := createSpinner(cmd.OutOrStdout(), fmt.Sprintf("%s%s", fullLineMessage.Message, spacerText))
	if err != nil {
		return errors.Wrap(err, "Could not create progressive output.")
	}

	ticker = time.NewTicker(HowFrequentlyToUpdateTicker)

	// Capture Ctrl+C if the user wants to finish early the job
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, ShutdownSignals...)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	finishedRunning := false
	cmdShuttingDown := false
	var returnError error
	returnError = nil

	// goroutine for handling spinner ticks and spinner completion messages
	go func() {
		for {
			log.Ctx(ctx).Trace().Msgf("Ticker goreturn")

			select {
			case <-tickerDone:
				ticker.Stop()
				log.Ctx(ctx).Trace().Msgf("Ticker goreturn done")
				return
			case t := <-ticker.C:
				if !quiet {
					fullLineMessage.TimerString = spinnerFmtDuration(t.Sub(j.Metadata.CreatedAt))
					spin.Message(fmt.Sprintf("%s %s", spacerText, fullLineMessage.TimerString))
					spin.StopMessage(fullLineMessage.PrintDone())
				}
			}
		}
	}()

	// goroutine for handling SIGINT from the signal channel, or context
	// completion messages.
	go func() {
		log.Ctx(ctx).Trace().Msgf("Signal goreturn")

		select {
		case s := <-signalChan: // first signal, cancel context
			log.Ctx(ctx).Debug().Msgf("Captured %v. Exiting...", s)
			if s == os.Interrupt {
				// Stop the spinner and let the rest of WaitForJobAndPrintResultsToUser
				// know that we're going to shut down so that it doesn't try to
				// restart the spinner after it's displayed the last line of
				// output.
				_ = spin.Stop()
				cmdShuttingDown = true

				// If finishedRunning is true, then we go term signal
				// because the loop finished normally.
				if !finishedRunning {
					if !quiet {
						cmd.Println("\n\n\rPrintout canceled (the job is still running).")
						cmd.Println(getMoreInfoString)
					}
					returnError = fmt.Errorf(PrintoutCanceledButRunningNormally)
				}
			} else {
				cmd.Println("Unexpected signal received. Exiting.")
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	// Loop through the events, printing those that are interesting, and then
	// shutting down when a this job reaches a terminal state.
	for {
		if !quiet && !cmdShuttingDown {
			if spin.Status().String() != "running" {
				err = spin.Start()
				if err != nil {
					return errors.Wrap(err, "Could not start spinner.")
				}
			}
		}

		log.Ctx(ctx).Trace().Msgf("Job Events:")
		for i := range jobEvents {
			log.Ctx(ctx).Trace().Msgf("\t%s - %s - %s",
				jobEvents[i].NewState,
				jobEvents[i].Time.UTC().String(),
				jobEvents[i].Comment)
		}
		log.Ctx(ctx).Trace().Msgf("\n")

		if err != nil {
			if _, ok := err.(*bacerrors.JobNotFound); ok {
				Fatal(cmd, fmt.Sprintf(`Somehow even though we submitted a job successfully,
											we were not able to get its status. ID: %s`, j.Metadata.ID), 1)
			} else {
				Fatal(cmd, fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", j.Metadata.ID, err), 1)
			}
		}

		if !quiet {
			for i := range jobEvents {
				if jobEvents[i].Type == model.JobHistoryTypeExecutionLevel {
					printingUpdateForEvent(cmd,
						&printedEventsTracker,
						jobEvents[i].NewStateType,
						spin)
				}
			}
		}

		// TODO: #1070 We should really streamline these two loops - when we get to a client side statemachine, that should take care of lots
		// Look for any terminal event in all the events. If it's done, we're done.
		for i := range jobEvents {
			// TODO: #837 We should be checking for the last event of a given type, not the first, across all shards.
			if eventsWorthPrinting[jobEvents[i].NewStateType].IsTerminal {
				// Send a signal to the goroutine that is waiting for Ctrl+C
				finishedRunning = true

				_ = spin.Stop()
				tickerDone <- true
				signalChan <- os.Interrupt
				return err
			}
		}

		if condition := ctx.Err(); condition != nil {
			signalChan <- os.Interrupt
			break
		} else {
			jobEvents, err = GetAPIClient().GetEvents(ctx, j.Metadata.ID)
			if err != nil {
				if _, ok := err.(*bacerrors.ContextCanceledError); ok {
					// We're done, the user canceled the job
					break
				} else {
					return errors.Wrap(err, "Error getting job events")
				}
			}
		}

		time.Sleep(time.Duration(500) * time.Millisecond) //nolint:gomnd // 500ms sleep
	} // end for

	return returnError
}

// Create a lock for printing events
var printedEventsLock sync.Mutex

func printingUpdateForEvent(cmd *cobra.Command, pe *sync.Map,
	jet model.ExecutionStateType,
	spin *yacspin.Spinner) bool {
	// We need to lock this because we're using a map
	printedEventsLock.Lock()
	defer printedEventsLock.Unlock()

	// We control all events being loaded, if nothing loads, something is seriously wrong.
	anyEvent, _ := pe.Load(jet)
	e := anyEvent.(printedEvents)

	// If it hasn't been printed yet, we'll print this event.
	// We'll also skip lines where there's no message to print.
	if eventsWorthPrinting[jet].Message != "" && !e.printed {
		e.printed = true
		pe.Store(jet, e)

		_ = spin.Pause()

		// Need to skip printing the initial submission event
		cmd.Printf("\r\033[K\r")
		if eventsWorthPrinting[jet].IsError {
			cmd.Printf("%s\n", fullLineMessage.PrintError())
		} else {
			cmd.Printf("%s\n", fullLineMessage.PrintDone())
		}

		if eventsWorthPrinting[jet].IsTerminal {
			cmd.Printf("\n%s\n", eventsWorthPrinting[jet].Message)
			return eventsWorthPrinting[jet].PrintDownload
		}

		fullLineMessage.Message = formatMessage(eventsWorthPrinting[jet].Message)

		spin.Prefix(fmt.Sprintf("%s %s", fullLineMessage.Message, spacerText))

		// start animating the spinner
		_ = spin.Unpause()
	}

	return eventsWorthPrinting[jet].PrintDownload
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

var spinnerEmoji = []string{"🐟", "🐠", "🐡"}

func createSpinner(w io.Writer, startingMessage string) (*yacspin.Spinner, error) {
	var spinnerCharSet []string
	for _, emoji := range spinnerEmoji {
		for i := 0; i < fullLineMessage.Width; i++ {
			spinnerCharSet = append(spinnerCharSet, fmt.Sprintf("%s%s%s",
				strings.Repeat(" ", fullLineMessage.Width-i),
				emoji,
				strings.Repeat(" ", i)))
		}
	}

	cfg := yacspin.Config{
		Frequency: 100 * time.Millisecond,
		CharSet:   spinnerCharSet,
		Writer:    w,
		// Have to set the Prefix on creation because
		// sometimes the spinner starts faster than the first print
		Prefix: startingMessage,
	}

	s, err := yacspin.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate spinner from methods: %v", err)
	}

	if err := s.CharSet(spinnerCharSet); err != nil {
		return nil, fmt.Errorf("failed to set charset: %v", err)
	}

	return s, nil
}

func spinnerFmtDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)

	min := (d % time.Hour) / time.Minute
	sec := (d % time.Minute) / time.Second
	ms := (d % time.Second) / time.Millisecond / 100

	minString, secString, msString := "", "", ""
	if min > 0 {
		minString = fmt.Sprintf("%02dm", min)
		secString = fmt.Sprintf("%02d", sec)
		msString = fmt.Sprintf(".%01ds", ms)
	} else if sec > 0 {
		secString = fmt.Sprintf("%01d", sec)
		msString = fmt.Sprintf(".%01ds", ms)
	} else {
		msString = fmt.Sprintf("0.%01ds", ms)
	}
	// If hour string exists, set it
	return fmt.Sprintf("%s%s%s", minString, secString, msString)
}

func formatMessage(msg string) string {
	maxLength := 0
	for _, v := range eventsWorthPrinting {
		if len(v.Message) > maxLength {
			maxLength = len(v.Message)
		}
	}

	return fmt.Sprintf("\t%s%s",
		strings.Repeat(" ", maxLength-len(msg)+2), msg)
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
