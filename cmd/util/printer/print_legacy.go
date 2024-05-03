package printer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels/legacymodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var eventsWorthPrintingLegacy = map[model.JobStateType]eventStruct{
	model.JobStateNew:        {Message: "Creating job for submission", IsTerminal: false, IsError: false},
	model.JobStateQueued:     {Message: "Job waiting to be scheduled", IsTerminal: false, IsError: false},
	model.JobStateInProgress: {Message: "Job in progress", IsTerminal: false, IsError: false},
	model.JobStateError:      {Message: "Error while executing the job", IsTerminal: true, IsError: true},
	model.JobStateCancelled:  {Message: "Job canceled", IsTerminal: true, IsError: false},
	model.JobStateCompleted:  {Message: "Job finished", IsTerminal: true, IsError: false},
}

// PrintJobExecutionLegacy displays information about the execution of a job
// TODO gocyclo rates this as a 24, which is very high, we should refactor someday.
//
//nolint:gocyclo,funlen
func PrintJobExecutionLegacy(
	ctx context.Context,
	j *model.Job,
	cmd *cobra.Command,
	downloadSettings *cliflags.DownloaderSettings,
	runtimeSettings *cliflags.RunTimeSettings,
	client *client.APIClient,
) error {
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

	if runtimeSettings.Follow {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", j.Metadata.ID)
		cmd.Printf("Waiting for logs... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")

		// Wait until the job has actually been accepted and started, otherwise this will fail waiting for
		// the execution to appear.
		for i := 0; i < 10; i++ {
			jobState, stateErr := client.GetJobState(ctx, j.ID())
			if stateErr != nil {
				return fmt.Errorf("failed waiting for execution to start: %w", stateErr)
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

		return util.Logs(cmd, util.LogOptions{
			JobID:  j.ID(),
			Follow: true,
		})
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := runtimeSettings.PrintJobIDOnly

	jobErr := WaitForJobAndPrintResultsToUser(ctx, cmd, j, quiet)
	if jobErr != nil {
		if jobErr.Error() == PrintoutCanceledButRunningNormally {
			return nil
		} else {
			cmd.PrintErrf("\nError submitting job: %s", jobErr)
		}
	}

	jobReturn, found, err := client.Get(ctx, j.Metadata.ID)
	if err != nil {
		return fmt.Errorf("error getting job: %w", err)
	}
	if !found {
		return fmt.Errorf("weird. Just ran the job, but we couldn't find it. Should be impossible. ID: %s", j.Metadata.ID)
	}

	js, err := client.GetJobState(ctx, jobReturn.Job.Metadata.ID)
	if err != nil {
		return fmt.Errorf("error getting job state: %w", err)
	}

	if runtimeSettings.PrintNodeDetails || jobErr != nil {
		cmd.Println("\nJob Results By Node:")
		for message, nodes := range summariseExecutionsLegacy(js) {
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
		cmd.Printf("\nTo download the results, execute:\n\t"+os.Args[0]+" get %s\n", j.ID())
	}

	if !quiet {
		cmd.Printf("\nTo get more details about the run, execute:\n\t"+os.Args[0]+" describe %s\n", j.ID())
	}

	if hasResults && runtimeSettings.AutoDownloadResults {
		if err := util.DownloadResultsHandler(ctx, cmd, j.Metadata.ID, downloadSettings); err != nil {
			return err
		}
	}
	return nil
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
	for _, v := range eventsWorthPrintingLegacy {
		widestString = math.Max(widestString, len(v.Message))
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

	var lastSeenTimestamp int64 = 0
	var lastEventState model.JobStateType
	for !cmdShuttingDown {
		// Get the job level history events that happened since the last one we saw
		jobEvents, err := util.GetAPIClient(ctx).GetEvents(ctx, j.Metadata.ID, legacymodels.EventFilterOptions{
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
				if !wasPrinted && eventsWorthPrintingLegacy[jet].Message != "" {
					printedEventsTracker[jet] = true

					// We shouldn't do anything with execution errors because there could
					// be retries following, so for now we will
					if !eventsWorthPrintingLegacy[jet].IsError && !eventsWorthPrintingLegacy[jet].IsTerminal {
						spinner.NextStep(eventsWorthPrintingLegacy[jet].Message)
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

// Groups the executions in the job state, returning a map of printable messages
// to node(s) that generated that message.
func summariseExecutionsLegacy(state model.JobState) map[string][]string {
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
			results[message] = append(results[message], idgen.ShortNodeID(execution.NodeID))
		}
	}
	return results
}
