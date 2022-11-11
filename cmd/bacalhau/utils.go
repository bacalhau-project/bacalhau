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
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/Masterminds/semver"
	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
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
	// what permissions do we give to a folder we create when downloading results
	AutoDownloadFolderPerm                    = 0755
	HowFrequentlyToUpdateTicker               = 50 * time.Millisecond
	DefaultTimeout              time.Duration = requesternode.DefaultJobExecutionTimeout
)

var eventsWorthPrinting = map[model.JobEventType]eventStruct{
	// In Rough execution order
	model.JobEventInitialSubmission: {Message: "Communicating with the network", IsTerminal: false, PrintDownload: true},

	model.JobEventCreated: {Message: "Creating job for submission", IsTerminal: false, PrintDownload: true},

	// Job is on Requester
	model.JobEventBid: {Message: "Finding node(s) for the job", IsTerminal: false, PrintDownload: true},

	// Job is on ComputeNode
	model.JobEventBidAccepted: {Message: "Running the job", IsTerminal: false, PrintDownload: true},
	model.JobEventRunning:     {Message: "Node started running the job", IsTerminal: false, PrintDownload: true},

	// Need to add a carriage return to the end of the line, but only this one
	model.JobEventComputeError: {Message: "Error while executing the job.", IsTerminal: true, PrintDownload: false},

	// Job is on StorageNode
	model.JobEventResultsProposed:  {Message: "Job finished, verifying results", IsTerminal: false, PrintDownload: true},
	model.JobEventResultsRejected:  {Message: "Results failed verification.", IsTerminal: true, PrintDownload: false},
	model.JobEventResultsAccepted:  {Message: "Results accepted, publishing", IsTerminal: false, PrintDownload: true},
	model.JobEventResultsPublished: {Message: "", IsTerminal: true, PrintDownload: true},

	// General Error?
	model.JobEventError: {Message: "Unknown error while running job.", IsTerminal: true, PrintDownload: false},

	// Should we print at all? Empty events get skipped
	model.JobEventBidCancelled: {},
	model.JobEventBidRejected:  {},
	model.JobEventDealUpdated:  {},
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

func GetAPIClient() *publicapi.APIClient {
	return publicapi.NewAPIClient(fmt.Sprintf("http://%s:%d", apiHost, apiPort))
}

// ensureValidVersion checks that the server version is the same or less than the client version
func ensureValidVersion(_ context.Context, clientVersion, serverVersion *model.BuildVersionInfo) error {
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

func NewIPFSDownloadFlags(settings *ipfs.IPFSDownloadSettings) *pflag.FlagSet {
	flags := pflag.NewFlagSet("IPFS Download flags", pflag.ContinueOnError)
	flags.IntVar(&settings.TimeoutSecs, "download-timeout-secs",
		settings.TimeoutSecs, "Timeout duration for IPFS downloads.")
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

func processDownloadSettings(settings ipfs.IPFSDownloadSettings, jobID string) (ipfs.IPFSDownloadSettings, error) {
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

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	cm *system.CleanupManager,
	cmd *cobra.Command,
	j *model.Job,
	runtimeSettings RunTimeSettings,
	downloadSettings ipfs.IPFSDownloadSettings,
	buildContext *bytes.Buffer,
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

	err := job.VerifyJob(ctx, j)
	if err != nil {
		log.Err(err).Msg("Job failed to validate.")
		return err
	}

	j, err = submitJob(ctx, apiClient, j, buildContext)
	if err != nil {
		return err
	}

	// if we are in --wait=false - print the id then exit
	// because all code after this point is related to
	// "wait for the job to finish" (via WaitAndPrintResultsToUser)
	if !runtimeSettings.WaitForJobToFinish {
		cmd.Print(j.ID + "\n")
		return nil
	}

	// if we are in --id-only mode - print the id
	if runtimeSettings.PrintJobIDOnly {
		cmd.Print(j.ID + "\n")
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := runtimeSettings.PrintJobIDOnly

	printDownload, err := WaitAndPrintResultsToUser(ctx, j, quiet)
	if err != nil {
		if err.Error() == PrintoutCanceledButRunningNormally {
			Fatal("", 0)
		} else {
			Fatal(fmt.Sprintf("Error submitting job: %s", err), 1)
		}
	}

	jobReturn, found, err := apiClient.Get(ctx, j.ID)
	if err != nil {
		Fatal(fmt.Sprintf("Error getting job: %s", err), 1)
	}
	if !found {
		Fatal(fmt.Sprintf("Weird. Just ran the job, but we couldn't find it. Should be impossible. ID: %s", j.ID), 1)
	}

	js, err := apiClient.GetJobState(ctx, jobReturn.ID)
	if err != nil {
		Fatal(fmt.Sprintf("Error getting job state: %s", err), 1)
	}

	// Need to create index because map ordering are not guaranteed
	nodeIndexes := make([]string, 0, len(js.Nodes))
	for i := range js.Nodes {
		nodeIndexes = append(nodeIndexes, i)
	}
	sort.Strings(nodeIndexes)

	printOut := "%s" // We only know this at the end, we'll fill it in there.
	resultsCID := ""
	indentOne := "  "
	indentTwo := strings.Repeat(indentOne, 2)
	if runtimeSettings.PrintNodeDetails {
		printOut += "\n"
		printOut += "Job Results By Node:\n"
		for i := range nodeIndexes {
			n := js.Nodes[nodeIndexes[i]]
			printOut += fmt.Sprintf("Node %s:\n", nodeIndexes[i][:8])
			for j, s := range n.Shards { //nolint:gocritic // very small loop, ok to be costly
				printOut += fmt.Sprintf(indentOne+"Shard %d:\n", j)
				printOut += fmt.Sprintf(indentTwo+"State: %s\n", s.State)
				printOut += fmt.Sprintf(indentTwo+"Status: %s\n", s.State)
				if s.RunOutput == nil {
					printOut += fmt.Sprintf(indentTwo + "No RunOutput for this shard\n")
				} else {
					printOut += fmt.Sprintf(indentTwo+"Container Exit Code: %d\n", s.RunOutput.ExitCode)
					resultsCID = s.PublishedResult.CID // They're all the same, doesn't matter if we assign it many times
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
					printResults("Stdout", s.RunOutput.STDOUT, s.RunOutput.StdoutTruncated)
					printResults("Stderr", s.RunOutput.STDERR, s.RunOutput.StderrTruncated)
				}
			}
		}
	}

	if printDownload {
		printOut += fmt.Sprintf(`
To download the results, execute:
%sbacalhau get %s
`, indentOne, j.ID)
	}

	printOut += fmt.Sprintf(`
To get more details about the run, execute:
%sbacalhau describe %s
`, indentOne, j.ID)

	// Have to do a final Sprintf so we can inject the resultsCID into the right place
	if resultsCID != "" {
		resultsCID = fmt.Sprintf("Results CID: %s\n", resultsCID)
	}
	if !quiet {
		RootCmd.Print(fmt.Sprintf(printOut, resultsCID))
	}

	if runtimeSettings.AutoDownloadResults {
		err = downloadResultsHandler(
			ctx,
			cm,
			cmd,
			j.ID,
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
	downloadSettings ipfs.IPFSDownloadSettings,
) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "Fetching results of job '%s'...\n", jobID)
	j, _, err := GetAPIClient().Get(ctx, jobID)

	if err != nil {
		if _, ok := err.(*bacerrors.JobNotFound); ok {
			return err
		} else {
			Fatal(fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", jobID, err), 1)
		}
	}

	results, err := GetAPIClient().GetResults(ctx, j.ID)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}

	processedDownloadSettings, err := processDownloadSettings(downloadSettings, j.ID)
	if err != nil {
		return err
	}

	err = ipfs.DownloadJob(
		ctx,
		cm,
		j.Spec.Outputs,
		results,
		processedDownloadSettings,
	)

	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Results for job '%s' have been written to...\n", jobID)
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", processedDownloadSettings.OutputDir)

	return nil
}

func submitJob(ctx context.Context,
	apiClient *publicapi.APIClient,
	j *model.Job,
	buildContext *bytes.Buffer,
) (*model.Job, error) {
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/utils.submitJob")
	defer span.End()

	j, err := apiClient.Submit(ctx, j, buildContext)
	if err != nil {
		return &model.Job{}, errors.Wrap(err, "failed to submit job")
	}
	return j, err
}

func ReadFromStdinIfAvailable(_ *cobra.Command, args []string) ([]byte, error) {
	if len(args) == 0 {
		r := bufio.NewReader(RootCmd.InOrStdin())
		var bytesResult []byte
		scanner := bufio.NewScanner(r)

		// buffered channel of dataStream
		dataStream := make(chan []byte, 1)

		// Run scanner.Bytes() function in it's own goroutine and pass back it's
		// response into dataStream channel.
		go func() {
			for scanner.Scan() {
				dataStream <- scanner.Bytes()
			}
			close(dataStream)
		}()

		// Listen on dataStream channel AND a timeout channel - which ever happens first.
		timedOut := false
		select {
		case res := <-dataStream:
			bytesResult = append(bytesResult, res...)
		case <-time.After(time.Duration(10) * time.Millisecond): //nolint:gomnd // 10ms timeout
			timedOut = true
		}

		if timedOut {
			RootCmd.Println("No input provided, waiting ... (Ctrl+D to complete)")
		}
		for scanner.Scan() {
			bytesResult = append(bytesResult, scanner.Bytes()...)
		}

		return bytesResult, nil
	}
	return nil, fmt.Errorf("should not be possible, args should be empty")
}

// Need these as global so that multiple routines can access
var fullLineMessage = ""
var currentLineMessage = ""
var stopMessage = ""
var timerMessage = ""
var doneMessage = " ✅"
var width = 6

const spacerText = " ... "

var ticker *time.Ticker
var tickerDone = make(chan bool)

//nolint:gocyclo,funlen // Better way to do this, Go doesn't have a switch on type
func WaitAndPrintResultsToUser(ctx context.Context, j *model.Job, quiet bool) (bool, error) {
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	if j == nil || j.ID == "" {
		return false, errors.New("No job returned from the server.")
	}
	getMoreInfoString := fmt.Sprintf(`
To get more information at any time, run:
   bacalhau describe %s`, j.ID)

	if !quiet {
		RootCmd.Printf("Job successfully submitted. Job ID: %s\n", j.ID)
		RootCmd.Printf("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n")
	}

	// Create a map of job state types to printed structs
	printedEventsTracker := make(map[model.JobEventType]*printedEvents)
	for _, jobEventType := range model.JobEventTypes() {
		printedEventsTracker[jobEventType] = &printedEvents{
			printed: false,
			order:   int(jobEventType),
		}
	}

	jobEvents, err := GetAPIClient().GetEvents(ctx, j.ID)
	if err != nil {
		Fatal(fmt.Sprintf("Failure retrieving job events '%s': %s\n", j.ID, err), 1)
	}

	// Inject "Job Initiated Event" to start - should we do this on the server?
	// TODO: #1068 Should jobs auto add a "start event" on the client at creation?
	jobEvents = append([]model.JobEvent{{EventName: model.JobEventInitialSubmission}}, jobEvents...)
	// Faking an initial time (sometimes it happens too fast to see)
	timerMessage = spinnerFmtDuration(30 * time.Millisecond) //nolint:gomnd // 30ms is just a default
	currentLineMessage = formatMessage(eventsWorthPrinting[model.JobEventInitialSubmission].Message)

	// Create a spinner var that will span all printouts
	spin, err := createSpinner(fmt.Sprintf("%s%s", currentLineMessage, spacerText))
	if err != nil {
		return false, errors.Wrap(err, "Could not create progressive output.")
	}

	ticker = time.NewTicker(HowFrequentlyToUpdateTicker)

	// Capture Ctrl+C if the user wants to finish early the job
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	finishedRunning := false
	var returnError error
	returnError = nil

	printDownloadFlag := true

	go func() {
		for {
			log.Trace().Msgf("Ticker goreturn")

			select {
			case <-tickerDone:
				return
			case t := <-ticker.C:
				timerMessage = spinnerFmtDuration(t.Sub(j.CreatedAt))
				spin.Message(fmt.Sprintf("%s%s", spacerText, timerMessage))
				fullLineMessage = fmt.Sprintf("%s %s %s %s",
					currentLineMessage,
					stopMessage,
					timerMessage,
					doneMessage)
				spin.StopMessage(fullLineMessage)
			}
		}
	}()

	go func() {
		log.Trace().Msgf("Ticker goreturn")

		select {
		case s := <-signalChan: // first signal, cancel context
			log.Debug().Msgf("Captured %v. Exiting...", s)
			if s == os.Interrupt {
				// If finishedRunning is true, then we go term signal
				// because the loop finished normally.
				if !finishedRunning {
					if !quiet {
						RootCmd.Println("\n\n\rPrintout canceled (the job is still running).")
						RootCmd.Println(getMoreInfoString)
					}
					returnError = fmt.Errorf(PrintoutCanceledButRunningNormally)
				}
			} else {
				RootCmd.Println("Unexpected signal received. Exiting.")
			}
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	if len(jobEvents) != 0 {
		for {
			if spin.Status().String() != "running" {
				err = spin.Start()
				if err != nil {
					return false, errors.Wrap(err, "Could not start spinner.")
				}
			}

			log.Trace().Msgf("Job Events:")
			for i := range jobEvents {
				log.Trace().Msgf("\t%s - %s - %s",
					model.GetStateFromEvent(jobEvents[i].EventName),
					jobEvents[i].EventTime.UTC().String(),
					jobEvents[i].EventName)
			}
			log.Trace().Msgf("\n")

			if err != nil {
				if _, ok := err.(*bacerrors.JobNotFound); ok {
					Fatal(fmt.Sprintf("Somehow even though we submitted a job successfully, we were not able to get its status. ID: %s", j.ID), 1)
				} else {
					Fatal(fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", j.ID, err), 1)
				}
			}

			if !quiet {
				for i := range jobEvents {
					// Will get overridden by the last event
					printDownloadFlag = printingUpdateForEvent(printedEventsTracker, jobEvents[i].EventName, spin)
				}
			}

			// TODO: #1070 We should really streamline these two loops - when we get to a client side statemachine, that should take care of lots
			// Look for any terminal event in all the events. If it's done, we're done.
			for i := range jobEvents {
				// TODO: #837 We should be checking for the last event of a given type, not the first, across all shards.
				if eventsWorthPrinting[jobEvents[i].EventName].IsTerminal {
					// Send a signal to the goroutine that is waiting for Ctrl+C
					finishedRunning = true

					if printDownloadFlag {
						_ = spin.Stop()
					} else {
						_ = spin.StopFail()
					}
					ticker.Stop()
					signalChan <- syscall.SIGINT
					return printDownloadFlag, err
				}
			}

			// Second loop we should streamline - see above
			jobEvents, err = GetAPIClient().GetEvents(ctx, j.ID)
			if err != nil {
				if _, ok := err.(*bacerrors.ContextCanceledError); ok {
					// Print out final message
					fullLineMessage = fmt.Sprintf("%s %s %s %s",
						currentLineMessage,
						stopMessage,
						timerMessage,
						doneMessage)

					// We're done, the user canceled the job
					_ = spin.Stop()
					ticker.Stop()
					signalChan <- syscall.SIGINT
					return false, nil
				} else {
					return false, errors.Wrap(err, "Error getting job events")
				}
			}

			time.Sleep(time.Duration(500) * time.Millisecond) //nolint:gomnd // 500ms sleep
		} // end for
	}

	return printDownloadFlag, returnError
}

func printingUpdateForEvent(pe map[model.JobEventType]*printedEvents,
	jet model.JobEventType,
	spin *yacspin.Spinner) bool {
	// If it hasn't been printed yet, we'll print this event.
	// We'll also skip lines where there's no message to print.
	if eventsWorthPrinting[jet].Message != "" && !pe[jet].printed {
		_ = spin.Pause()

		// log.Debug().Msgf("Printing event: %s\n", jet)
		RootCmd.Printf("\r\033[K\r")
		RootCmd.Printf("%s\n", fullLineMessage)

		if eventsWorthPrinting[jet].IsTerminal {
			RootCmd.Printf("\n%s\n", eventsWorthPrinting[jet].Message)
			pe[jet].printed = true
			return eventsWorthPrinting[jet].PrintDownload
		}

		currentLineMessage = formatMessage(eventsWorthPrinting[jet].Message)

		// Need to do a full line message update here too - incase we do two events with no tick between them
		fullLineMessage = fmt.Sprintf("%s %s %s %s",
			currentLineMessage,
			stopMessage,
			timerMessage,
			doneMessage)
		spin.Prefix(fmt.Sprintf("%s%s", currentLineMessage, spacerText))
		spin.StopMessage(fullLineMessage)

		// start animating the spinner
		_ = spin.Unpause()

		pe[jet].printed = true
	}

	return eventsWorthPrinting[jet].PrintDownload
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
	b, _ := model.JSONMarshalWithMax(c)
	RootCmd.Println(string(b))
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

func createSpinner(startingMessage string) (*yacspin.Spinner, error) {
	var spinnerCharSet []string
	for _, emoji := range spinnerEmoji {
		for i := 0; i < width; i++ {
			spinnerCharSet = append(spinnerCharSet, fmt.Sprintf("%s%s%s",
				strings.Repeat(" ", width-i),
				emoji,
				strings.Repeat(" ", i)))
		}
	}

	cfg := yacspin.Config{
		Frequency: 100 * time.Millisecond,
		CharSet:   spinnerCharSet,
		Writer:    RootCmd.OutOrStdout(),
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

	// Need to add 8 to have everything line up.
	stopMessage = strings.Repeat(".", width+10) //nolint:gomnd // extra spacing

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
