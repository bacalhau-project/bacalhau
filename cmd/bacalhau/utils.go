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
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Masterminds/semver"
	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	JSONFormat                         string = "json"
	YAMLFormat                         string = "yaml"
	DefaultDockerRunWaitSeconds               = 600
	PrintoutCanceledButRunningNormally string = "printout canceled but running normally"
)

var eventsWorthPrinting = map[model.JobEventType]eventStruct{
	// In Rough execution order
	model.JobEventCreated: {Message: "Creating job for submission", IsTerminal: false},

	// Job is on Requester
	model.JobEventBid:         {Message: "Finding node(s) for the job", IsTerminal: false},
	model.JobEventBidAccepted: {Message: "Node accepted the job", IsTerminal: false},

	// Job is on ComputeNode
	model.JobEventRunning: {Message: "Node started running the job", IsTerminal: false},

	// Need to add a carriage return to the end of the line, but only this one
	model.JobEventComputeError: {Message: "Error while executing the job.\n", IsTerminal: true},

	// Job is on StorageNode
	model.JobEventResultsProposed:  {Message: "Job finished, verifying results", IsTerminal: false},
	model.JobEventResultsRejected:  {Message: "Results failed verification.", IsTerminal: true},
	model.JobEventResultsAccepted:  {Message: "Results accepted, publishing", IsTerminal: false},
	model.JobEventResultsPublished: {Message: "", IsTerminal: true},

	// General Error?
	model.JobEventError: {Message: "Unknown error while running job.", IsTerminal: true},

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
	Message    string
	IsTerminal bool
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
	AutoDownloadResults   bool // Automatically download the results after finishing
	IPFSGetTimeOut        int  // Timeout for IPFS in seconds
	IsLocal               bool // Job should be executed locally
	WaitForJobToFinish    bool // Wait for the job to finish before returning
	WaitForJobTimeoutSecs int  // Timeout for waiting for the job to finish
}

func NewRunTimeSettings() *RunTimeSettings {
	return &RunTimeSettings{
		AutoDownloadResults:   false,
		WaitForJobToFinish:    true,
		WaitForJobTimeoutSecs: DefaultDockerRunWaitSeconds,
		IPFSGetTimeOut:        10,
		IsLocal:               false,
	}
}

func setupRunTimeFlags(cmd *cobra.Command, settings *RunTimeSettings) {
	cmd.PersistentFlags().IntVarP(
		&settings.IPFSGetTimeOut, "gettimeout", "g", settings.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`,
	)

	cmd.PersistentFlags().BoolVar(
		&settings.IsLocal, "local", settings.IsLocal,
		`Run the job locally. Docker is required`,
	)

	cmd.PersistentFlags().BoolVar(
		&settings.WaitForJobToFinish, "wait", settings.WaitForJobToFinish,
		`Wait for the job to finish.`,
	)

	cmd.PersistentFlags().IntVar(
		&settings.WaitForJobTimeoutSecs, "wait-timeout-secs", settings.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`,
	)
}

//nolint:funlen,gocyclo // Refactor later
func ExecuteJob(ctx context.Context,
	cm *system.CleanupManager,
	cmd *cobra.Command,
	j *model.Job,
	runtimeSettings RunTimeSettings,
	downloadSettings ipfs.IPFSDownloadSettings,
	idOnly bool,
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

	j, err = submitJob(ctx, apiClient, j)
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
	if idOnly {
		cmd.Print(j.ID + "\n")
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := idOnly

	err = WaitAndPrintResultsToUser(ctx, j, quiet)
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
	printOut += "Job Results By Node:\n"
	indentOne := "  "
	indentTwo := strings.Repeat(indentOne, 2)
	resultsCID := ""
	for i := range nodeIndexes {
		n := js.Nodes[nodeIndexes[i]]
		printOut += fmt.Sprintf("Node %s:\n", nodeIndexes[i][:8])
		for j, s := range n.Shards { //nolint:gocritic // very small loop, ok to be costly
			printOut += fmt.Sprintf(indentOne+"Shard %d:\n", j)
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

	printOut += fmt.Sprintf(`
To download the results, execute:
%sbacalhau get %s

To get more details about the run, execute:
%sbacalhau describe %s
`, indentOne, j.ID, indentOne, j.ID)

	// Have to do a final Sprintf so we can inject the resultsCID into the right place
	if resultsCID != "" {
		resultsCID = fmt.Sprintf("Results CID: %s\n", resultsCID)
	}
	if !quiet {
		RootCmd.Print(fmt.Sprintf(printOut, resultsCID))
	}

	if runtimeSettings.AutoDownloadResults {
		results, err := getResults(ctx, apiClient, j)
		if err != nil {
			return errors.Wrap(err, "error getting results")
		}

		if len(results) == 0 {
			return fmt.Errorf("no results found")
		}

		err = downloadResults(ctx, cmd, cm, j.Spec.Outputs, results, downloadSettings)
		if err != nil {
			return errors.Wrap(err, "error downloading results")
		}
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
	outputs []model.StorageSpec,
	results []model.StorageSpec,
	downloadSettings ipfs.IPFSDownloadSettings) error {
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/utils.downloadresults")
	defer span.End()

	err := ipfs.DownloadJob(
		ctx,
		cm,
		outputs,
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

//nolint:gocyclo,funlen // Better way to do this, Go doesn't have a switch on type
func WaitAndPrintResultsToUser(ctx context.Context, j *model.Job, quiet bool) error {
	if j == nil || j.ID == "" {
		return errors.New("No job returned from the server.")
	}
	getMoreInfoString := fmt.Sprintf(`
To get more information at any time, run:
   bacalhau describe %s`, j.ID)

	if !quiet {
		RootCmd.Printf("Job successfully submitted. Job ID: %s\n", j.ID)
		RootCmd.Printf("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
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

	go func() {
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
			log.Debug().Msgf("Job Events:")
			for i := range jobEvents {
				log.Debug().Msgf("\t%s - %s - %s",
					model.GetStateFromEvent(jobEvents[i].EventName),
					jobEvents[i].EventTime.UTC().String(),
					jobEvents[i].EventName)
			}
			log.Debug().Msgf("\n")

			if err != nil {
				if _, ok := err.(*bacerrors.JobNotFound); ok {
					Fatal(fmt.Sprintf("Somehow even though we submitted a job successfully, we were not able to get its status. ID: %s", j.ID), 1)
				} else {
					Fatal(fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", j.ID, err), 1)
				}
			}

			if !quiet {
				for i := range jobEvents {
					printingUpdateForEvent(printedEventsTracker, jobEvents[i].EventName)
				}
			}

			// Look for any terminal event in all the events. If it's done, we're done.
			for i := range jobEvents {
				// TODO: #837 We should be checking for the last event of a given type, not the first, across all shards.
				if eventsWorthPrinting[jobEvents[i].EventName].IsTerminal {
					// Send a signal to the goroutine that is waiting for Ctrl+C
					finishedRunning = true
					signalChan <- syscall.SIGINT
					break
				}
			}

			if condition := ctx.Err(); condition != nil {
				signalChan <- syscall.SIGINT
				break
			} else {
				jobEvents, err = GetAPIClient().GetEvents(ctx, j.ID)
				if err != nil {
					if _, ok := err.(*bacerrors.ContextCanceledError); ok {
						// We're done, the user canceled the job
						break
					} else {
						return errors.Wrap(err, "Error getting job events")
					}
				}
			}

			time.Sleep(time.Duration(1) * time.Millisecond) //nolint:gomnd // 500ms sleep
		} // end for
	}

	return returnError
}

func printingUpdateForEvent(pe map[model.JobEventType]*printedEvents, jet model.JobEventType) {
	maxLength := 0
	for _, v := range eventsWorthPrinting {
		if len(v.Message) > maxLength {
			maxLength = len(v.Message)
		}
	}

	// If it hasn't been printed yet, we'll print this event.
	// We'll also skip lines where there's no message to print.
	if eventsWorthPrinting[jet].Message != "" && !pe[jet].printed {
		// Only print " done" after the first line.
		firstLine := true
		for v := range pe {
			firstLine = firstLine && !pe[v].printed
		}
		if !firstLine {
			RootCmd.Println("done ✅")
		}

		RootCmd.Printf("\t%s%s",
			strings.Repeat(" ", maxLength-len(eventsWorthPrinting[jet].Message)+2),
			eventsWorthPrinting[jet].Message)
		if !eventsWorthPrinting[jet].IsTerminal {
			RootCmd.Print(" ... ")
		} else {
			RootCmd.Println()
		}
		pe[jet].printed = true
	}
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
