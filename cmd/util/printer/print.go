package printer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const PrintoutCanceledButRunningNormally string = "printout canceled but running normally"

var eventsWorthPrinting = map[models.JobStateType]eventStruct{
	models.JobStateTypePending:   {Message: "Creating job for submission", IsTerminal: false, IsError: false},
	models.JobStateTypeRunning:   {Message: "Job in progress", IsTerminal: false, IsError: false},
	models.JobStateTypeFailed:    {Message: "Error while executing the job", IsTerminal: true, IsError: true},
	models.JobStateTypeStopped:   {Message: "Job canceled", IsTerminal: true, IsError: false},
	models.JobStateTypeCompleted: {Message: "Job finished", IsTerminal: true, IsError: false},
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
	client *clientv2.Client,
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
		return followLogs(cmd, jobID, client)
	}

	// if we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := runtimeSettings.PrintJobIDOnly

	jobErr := waitForJobAndPrintResultsToUser(ctx, cmd, jobID, quiet, client)
	if jobErr != nil {
		if jobErr.Error() == PrintoutCanceledButRunningNormally {
			return nil
		} else {
			cmd.PrintErrf("\nError submitting job: %s", jobErr)
		}
	}

	if runtimeSettings.PrintNodeDetails || jobErr != nil {
		executions, err := client.Jobs().Executions(&apimodels.ListJobExecutionsRequest{
			JobID: jobID,
		})
		if err != nil {
			return fmt.Errorf("failed getting job executions: %w", err)
		}
		cmd.Println("\nJob Results By Node:")
		for message, nodes := range summariseExecutions(executions.Executions) {
			cmd.Printf("• Node %s: ", strings.Join(nodes, ", "))
			if strings.ContainsRune(message, '\n') {
				cmd.Printf("\n\t%s\n", strings.Join(strings.Split(message, "\n"), "\n\t"))
			} else {
				cmd.Println(message)
			}
		}
	}
	if !quiet {
		cmd.Println()
		cmd.Println("To get more details about the run, execute:")
		cmd.Println("\tbacalhau job describe " + jobID)

		cmd.Println()
		cmd.Println("To get more details about the run executions, execute:")
		cmd.Println("\tbacalhau job executions " + jobID)
	}

	return nil
}

func followLogs(cmd *cobra.Command, jobID string, client *clientv2.Client) error {
	cmd.Printf("Job successfully submitted. Job ID: %s\n", jobID)
	cmd.Printf("Waiting for logs... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")

	// Wait until the job has actually been accepted and started, otherwise this will fail waiting for
	// the execution to appear.
	for i := 0; i < 10; i++ {
		resp, err := client.Jobs().Get(&apimodels.GetJobRequest{
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

	return util.Logs(cmd, jobID, true, true)
}

// waitForJobAndPrintResultsToUser uses new job state  to decide what to output to the terminal
// using a spinner to show long-running tasks. When the job is complete (or the user
// triggers SIGINT) then the function will complete and stop outputting to the terminal.
//
//nolint:gocyclo,funlen
func waitForJobAndPrintResultsToUser(
	ctx context.Context, cmd *cobra.Command, jobID string, quiet bool, client *clientv2.Client) error {
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

	var lastEventState models.JobStateType
	for !cmdShuttingDown {
		resp, err := client.Jobs().Get(&apimodels.GetJobRequest{
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

		// Have we been cancel(l)ed?
		if condition := ctx.Err(); condition != nil {
			break
		}

		// TODO: add exponential backoff if there were no state updates
		time.Sleep(time.Duration(500) * time.Millisecond) //nolint:gomnd // 500ms sleep
	}

	return returnError
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
			results[message] = append(results[message], idgen.ShortID(execution.NodeID))
		}
	}
	return results
}
