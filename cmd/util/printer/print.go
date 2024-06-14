package printer

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mitchellh/go-wordwrap"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/term"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	libmath "github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const PrintoutCanceledButRunningNormally string = "printout canceled but running normally"

var eventsWorthPrinting = map[models.JobStateType]eventStruct{
	models.JobStateTypePending:   {Message: "Creating job for submission"},
	models.JobStateTypeRunning:   {Message: "Job in progress"},
	models.JobStateTypeFailed:    {Message: "Error while executing the job", IsTerminal: true, IsError: true},
	models.JobStateTypeStopped:   {Message: "Job canceled", IsTerminal: true},
	models.JobStateTypeCompleted: {Message: "Job finished", IsTerminal: true},
}

type eventStruct struct {
	Message    string
	IsTerminal bool
	IsError    bool
}

// PrintJobExecution displays information about the execution of a job
func PrintJobExecution(
	ctx context.Context,
	jobID string,
	cmd *cobra.Command,
	runtimeSettings *cliflags.RunTimeSettings,
	client clientv2.API,
) error {
	// if we are in --wait=false - print the id then exit
	// because all code after this point is related to
	// "wait for the job to finish" (via waitForJobAndPrintResultsToUser)
	if !runtimeSettings.WaitForJobToFinish {
		cmd.Print(jobID + "\n")
		return nil
	}

	// if we are in --id-only mode - print the id
	if runtimeSettings.PrintJobIDOnly {
		cmd.Print(jobID + "\n")
	}

	if runtimeSettings.Follow {
		return followLogs(cmd, client, jobID, client)
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := runtimeSettings.PrintJobIDOnly

	jobErr := waitForJobAndPrintResultsToUser(ctx, cmd, jobID, quiet, client)
	if jobErr != nil {
		if jobErr.Error() == PrintoutCanceledButRunningNormally {
			return nil
		}

		history, err := client.Jobs().History(ctx, &apimodels.ListJobHistoryRequest{
			JobID:     jobID,
			EventType: "execution",
		})
		if err != nil {
			return fmt.Errorf("failed getting job history: %w", err)
		}

		historySummary := summariseHistoryEvents(history.History)
		if len(historySummary) > 0 {
			for _, event := range historySummary {
				printEvent(cmd, event)
			}
		} else {
			printError(cmd, jobErr)
		}
	}

	if runtimeSettings.PrintNodeDetails {
		executions, err := client.Jobs().Executions(ctx, &apimodels.ListJobExecutionsRequest{
			JobID: jobID,
		})
		if err != nil {
			return fmt.Errorf("failed getting job executions: %w", err)
		}
		summary := summariseExecutions(executions.Executions)
		if len(summary) > 0 {
			cmd.Println("\nJob Results By Node:")
			for message, runs := range summary {
				nodes := len(lo.Uniq(runs))
				prefix := fmt.Sprintf("• Node %s: ", runs[0])
				if len(runs) > 1 {
					prefix = fmt.Sprintf("• %d runs on %d nodes: ", len(runs), nodes)
				}
				printIndentedString(cmd, prefix, strings.Trim(message, "\n"), none, 0)
			}
		} else {
			cmd.Println()
		}
	}
	if !quiet {
		cmd.Println()
		cmd.Println("To get more details about the run, execute:")
		cmd.Println("\t" + os.Args[0] + " job describe " + jobID)

		cmd.Println()
		cmd.Println("To get more details about the run executions, execute:")
		cmd.Println("\t" + os.Args[0] + " job executions " + jobID)
	}

	return nil
}

func followLogs(cmd *cobra.Command, api clientv2.API, jobID string, client clientv2.API) error {
	cmd.Printf("Job successfully submitted. Job ID: %s\n", jobID)
	cmd.Printf("Waiting for logs... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")

	// Wait until the job has actually been accepted and started, otherwise this will fail waiting for
	// the execution to appear.
	for i := 0; i < 10; i++ {
		resp, err := client.Jobs().Get(cmd.Context(), &apimodels.GetJobRequest{
			JobID: jobID,
		})
		if err != nil {
			return fmt.Errorf("failed getting job: %w", err)
		}
		if resp.Job.State.StateType != models.JobStateTypePending {
			break
		}
		// TODO: add exponential backoff if there were no state updates
		time.Sleep(time.Duration(1) * time.Second)
	}

	return util.Logs(cmd, api, util.LogOptions{
		JobID:  jobID,
		Follow: true,
	})
}

// waitForJobAndPrintResultsToUser uses new job state  to decide what to output to the terminal
// using a spinner to show long-running tasks. When the job is complete (or the user
// triggers SIGINT) then the function will complete and stop outputting to the terminal.
//
//nolint:gocyclo,funlen
func waitForJobAndPrintResultsToUser(
	ctx context.Context, cmd *cobra.Command, jobID string, quiet bool, client clientv2.API) error {
	getMoreInfoString := fmt.Sprintf(`
To get more information at any time, run:
   bacalhau job describe %s`, jobID)

	cancelString := fmt.Sprintf(`
To cancel the job, run:
   bacalhau job stop %s`, jobID)

	if !quiet {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", jobID)
		cmd.Printf("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
	}

	// Create a map of job state types -> boolean, this is a record of what has been printed
	// so far.
	printedEventsTracker := make(map[models.JobStateType]bool)
	for _, jobEventType := range models.JobStateTypes() {
		printedEventsTracker[jobEventType] = false
	}

	// Inject "Job Initiated Event" to start - should we do this on the server?
	// TODO: #1068 Should jobs auto add a "start event" on the client at creation?
	// Faking an initial time (sometimes it happens too fast to see)
	startMessage := "Communicating with the network"

	// Decide where the spinner should write it's output, by default we want stdout
	// but we can also disable output here if we have been asked to be quiet.
	writer := cmd.OutOrStdout()
	if quiet {
		writer = io.Discard
	}

	widestString := len(startMessage)
	for _, v := range eventsWorthPrinting {
		widestString = libmath.Max(widestString, len(v.Message))
	}

	spinner, err := NewSpinner(ctx, writer, widestString, false)
	if err != nil {
		return err
	}
	spinner.Run()
	spinner.NextStep(startMessage)

	// Capture Ctrl+C if the user wants to finish early the job
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, util.ShutdownSignals...)
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

	var lastEventState models.JobStateType
	for !cmdShuttingDown {
		resp, err := client.Jobs().Get(ctx, &apimodels.GetJobRequest{
			JobID: jobID,
		})
		if err != nil {
			if _, ok := err.(*bacerrors.ContextCanceledError); ok {
				// We're done, the user canceled the job
				cmdShuttingDown = true
				continue
			} else {
				return errors.Wrap(err, "Error getting job")
			}
		}

		jobState := resp.Job.State

		if !quiet {
			wasPrinted := printedEventsTracker[jobState.StateType]

			// If it hasn't been printed yet, we'll print this event.
			// We'll also skip lines where there's no message to print.
			if !wasPrinted && eventsWorthPrinting[jobState.StateType].Message != "" {
				printedEventsTracker[jobState.StateType] = true

				// We shouldn't do anything with execution errors because there could
				// be retries following, so for now we will
				if !eventsWorthPrinting[jobState.StateType].IsError && !eventsWorthPrinting[jobState.StateType].IsTerminal {
					spinner.NextStep(eventsWorthPrinting[jobState.StateType].Message)
				}
			} else if wasPrinted && lastEventState == jobState.StateType {
				spinner.msgMutex.Lock()
				spinner.msg.Detail = jobState.Message
				spinner.msgMutex.Unlock()
			}
		}

		lastEventState = jobState.StateType

		if resp.Job.IsTerminal() {
			if jobState.StateType != models.JobStateTypeCompleted {
				returnError = errors.New(jobState.Message)
				spinner.Done(StopFailed)
			} else {
				spinner.Done(StopSuccess)
			}
			cmdShuttingDown = true
			break
		}

		// If the job is long running, and it's running, we can stop the spinner
		if resp.Job.IsLongRunning() && resp.Job.State.StateType == models.JobStateTypeRunning {
			spinner.Done(StopSuccess)
			cmdShuttingDown = true
			break
		}

		// Have we been cancel(l)ed?
		if condition := ctx.Err(); condition != nil {
			break
		}

		// TODO: add exponential backoff if there were no state updates
		time.Sleep(time.Duration(500) * time.Millisecond) //nolint:gomnd // 500ms sleep
	}

	return returnError
}

var (
	none  = color.New(color.Reset)
	red   = color.New(color.FgRed)
	green = color.New(color.FgGreen)
)

const (
	errorPrefix = "Error: "
	hintPrefix  = "Hint: "
)

var terminalWidth int

func getTerminalWidth(cmd *cobra.Command) uint {
	if terminalWidth == 0 {
		var err error
		terminalWidth, _, err = term.GetSize(int(os.Stderr.Fd()))
		if err != nil || terminalWidth <= 0 {
			log.Ctx(cmd.Context()).Debug().Err(err).Msg("Failed to get terminal size")
			terminalWidth = math.MaxInt8
		}
	}
	return uint(terminalWidth)
}

func printEvent(cmd *cobra.Command, event models.Event) {
	printIndentedString(cmd, errorPrefix, event.Message, red, 0)
	if event.Details != nil && event.Details[models.DetailsKeyHint] != "" {
		printIndentedString(cmd, hintPrefix, event.Details[models.DetailsKeyHint], green, uint(len(errorPrefix)))
	}
}

func printError(cmd *cobra.Command, err error) {
	printIndentedString(cmd, errorPrefix, err.Error(), red, 0)
}

func printIndentedString(cmd *cobra.Command, prefix, msg string, prefixColor *color.Color, startIndent uint) {
	maxWidth := getTerminalWidth(cmd)
	blockIndent := int(startIndent) + len(prefix)
	blockTextWidth := maxWidth - startIndent - uint(len(prefix))

	cmd.PrintErrln()
	cmd.PrintErr(strings.Repeat(" ", int(startIndent)))
	prefixColor.Fprintf(cmd.ErrOrStderr(), prefix)
	for i, line := range strings.Split(wordwrap.WrapString(msg, blockTextWidth), "\n") {
		if i > 0 {
			cmd.PrintErr(strings.Repeat(" ", blockIndent))
		}
		cmd.PrintErrln(line)
	}
}

// Groups the executions in the job state, returning a map of printable messages
// to node(s) that generated that message.
func summariseExecutions(executions []*models.Execution) map[string][]string {
	results := make(map[string][]string, len(executions))
	for _, execution := range executions {
		var message string
		if execution.RunOutput != nil {
			if execution.RunOutput.ErrorMsg != "" {
				message = execution.RunOutput.ErrorMsg
			} else if execution.RunOutput.ExitCode > 0 {
				message = execution.RunOutput.STDERR
			} else {
				message = execution.RunOutput.STDOUT
			}
		} else if execution.IsDiscarded() {
			message = execution.ComputeState.Message
		}

		if message != "" {
			results[message] = append(results[message], idgen.ShortNodeID(execution.NodeID))
		}
	}
	return results
}

func summariseHistoryEvents(history []*models.JobHistory) []models.Event {
	slices.SortFunc(history, func(a, b *models.JobHistory) int {
		return a.Occurred().Compare(b.Occurred())
	})

	events := make(map[string]models.Event, len(history))
	for _, entry := range history {
		hasDetails := entry.Event.Details != nil
		failsExecution := hasDetails && entry.Event.Details[models.DetailsKeyFailsExecution] == "true"
		terminalState := entry.ExecutionState.New.IsTermainl()
		if (failsExecution || terminalState) && entry.Event.Message != "" {
			events[entry.Event.Message] = entry.Event
		}
	}

	return maps.Values(events)
}
