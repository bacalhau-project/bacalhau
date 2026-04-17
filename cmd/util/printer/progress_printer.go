package printer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
)

const progressTimeout = 5 * time.Minute

type JobProgressPrinter struct {
	client          clientv2.API
	runtimeSettings *cliflags.RunTimeSettings
}

func NewJobProgressPrinter(client clientv2.API, runtimeSettings *cliflags.RunTimeSettings) *JobProgressPrinter {
	return &JobProgressPrinter{
		client:          client,
		runtimeSettings: runtimeSettings,
	}
}

// PrintJobProgress displays the job progress based on CLI runtime settings
func (j *JobProgressPrinter) PrintJobProgress(ctx context.Context, job *models.Job, cmd *cobra.Command) error {
	// If we are in `--wait=false` print the job.ID and then exit.
	// All the code after this point is to show the progress of the job.
	if !j.runtimeSettings.WaitForJobToFinish {
		cmd.Print(job.ID + "\n")
		return nil
	}

	// If we are n `id-only` mode - print the id
	if j.runtimeSettings.PrintJobIDOnly {
		cmd.Print(job.ID + "\n")
	}

	// Follow Logs
	if j.runtimeSettings.Follow {
		return j.followLogs(ctx, job, cmd)
	}

	// Follow Progress
	if err := j.followProgress(ctx, job, cmd); err != nil {
		cmd.Println()
		PrintError(cmd, err)
	}

	if err := j.printNodeDetails(ctx, cmd, job); err != nil {
		return err
	}

	j.printJobDetailsInstructions(cmd, job)

	return nil
}

func (j *JobProgressPrinter) followLogs(ctx context.Context, job *models.Job, cmd *cobra.Command) error {
	if !j.isQuiet() {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", job.ID)
		cmd.Printf("Waiting for logs... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
	}

	// Wait until the job has actually been accepted and started, otherwise this will fail waiting for
	// the execution to appear.
	for i := 0; i < 10; i++ {
		resp, err := j.client.Jobs().Get(ctx, &apimodels.GetJobRequest{JobIDOrName: job.ID})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("failed getting job: %w", err)
		}
		if resp.Job.State.StateType != models.JobStateTypePending {
			break
		}
		// TODO: add exponential backoff if there were no state updates
		time.Sleep(time.Second)
	}

	return util.Logs(cmd, j.client, util.LogOptions{JobID: job.ID, Follow: true})
}

func (j *JobProgressPrinter) followProgress(ctx context.Context, job *models.Job, cmd *cobra.Command) error {
	ctx, cancel := context.WithTimeout(ctx, progressTimeout)
	defer cancel()

	if !j.isQuiet() {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", job.ID)
		cmd.Print("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
	}

	currentEventPrinter := j.createEventPrinter(cmd)

	eventChan := make(chan *models.JobHistory)
	errChan := make(chan error, 1)
	go j.fetchEvents(ctx, job, eventChan, errChan)

	// process events until the context is canceled or the job is done
	if err := j.handleEvents(ctx, currentEventPrinter, eventChan, errChan); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			j.printTimeoutMessage(cmd)
			return nil
		} else if errors.Is(err, context.Canceled) {
			j.printCancellationMessage(cmd)
			return nil
		}
		return err
	}

	return j.checkFinalJobState(ctx, job, cmd)
}

func (j *JobProgressPrinter) fetchEvents(ctx context.Context, job *models.Job, eventChan chan<- *models.JobHistory, errChan chan<- error) {
	defer close(eventChan)

	// Create a ticker that ticks every 500 milliseconds
	//nolint:mnd    // Time interval easier to read this way
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var nextToken string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			response, err := j.client.Jobs().History(ctx, &apimodels.ListJobHistoryRequest{
				JobIDOrName: job.ID,
				EventType:   "all",
				BaseVersionedListRequest: apimodels.BaseVersionedListRequest{
					JobVersion: job.Version,
					BaseListRequest: apimodels.BaseListRequest{
						NextToken: nextToken,
						Limit:     10,
					},
				},
			})
			if err != nil {
				errChan <- err
				return
			}

			for _, event := range response.Items {
				eventChan <- event
				// If the job is long running, we only care when an execution is created
				if job.IsLongRunning() && event.IsExecutionLevel() {
					return
				}
			}

			if response.NextToken == "" {
				return
			}
			nextToken = response.NextToken
		}
	}
}

func (j *JobProgressPrinter) handleEvents(
	ctx context.Context,
	printer eventPrinter,
	eventChan <-chan *models.JobHistory,
	errChan <-chan error,
) error {
	defer func() { _ = printer.close() }() // close the printer to clear any spinner before exiting

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventChan:
			if !ok {
				return nil // All events processed
			}
			if err := printer.printEvent(event); err != nil {
				return err
			}
		case err := <-errChan:
			return err
		}
	}
}

func (j *JobProgressPrinter) checkFinalJobState(ctx context.Context, job *models.Job, cmd *cobra.Command) error {
	resp, err := j.client.Jobs().Get(ctx, &apimodels.GetJobRequest{JobIDOrName: job.ID})
	if err != nil {
		return fmt.Errorf("failed getting job: %w", err)
	}
	if resp.Job.State.StateType == models.JobStateTypeFailed {
		return fmt.Errorf("job failed")
	}
	return nil
}

func (j *JobProgressPrinter) createEventPrinter(cmd *cobra.Command) eventPrinter {
	if j.isQuiet() {
		return newQuitePrinter(cmd)
	}
	if j.runtimeSettings.GroupEvents {
		return newGroupedEventPrinter(cmd)
	}
	return newSequentialEventPrinter(cmd)
}

func (j *JobProgressPrinter) printNodeDetails(ctx context.Context, cmd *cobra.Command, job *models.Job) error {
	if !j.runtimeSettings.PrintNodeDetails {
		return nil
	}

	executions, err := j.client.Jobs().Executions(ctx, &apimodels.ListJobExecutionsRequest{JobIDOrName: job.ID})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("failed getting job executions: %w", err)
	}

	summary := SummariseExecutions(executions.Items)
	if len(summary) > 0 {
		cmd.Println("\nJob Results By Node:")
		for message, runs := range summary {
			nodes := len(lo.Uniq(runs))
			prefix := fmt.Sprintf("• Node %s: ", runs[0])
			if len(runs) > 1 {
				prefix = fmt.Sprintf("• %d runs on %d nodes: ", len(runs), nodes)
			}
			printIndentedString(cmd, prefix, strings.TrimSpace(message), none, 0)
		}
	} else {
		cmd.Println()
	}

	return nil
}

func (j *JobProgressPrinter) printJobDetailsInstructions(cmd *cobra.Command, job *models.Job) {
	if j.isQuiet() {
		return
	}

	// query the server for the job spec to get any server side defaults and transformations,
	// such as if a default publisher was applied
	resp, err := j.client.Jobs().Get(cmd.Context(), &apimodels.GetJobRequest{JobIDOrName: job.ID})
	if err != nil {
		// just log and continue with the existing job details
		PrintWarning(cmd, fmt.Sprintf("Failed to get updated job details: %v", err))
	} else {
		job = resp.Job
	}

	cmd.Println()
	cmd.Println("To get more details about the run, execute:")
	cmd.Printf("\t%s job describe %s\n", os.Args[0], job.ID)

	cmd.Println()
	cmd.Println("To get more details about the run executions, execute:")
	cmd.Printf("\t%s job executions %s\n", os.Args[0], job.ID)

	// Only print help for downloading the job if it contained a publisher.
	if !lo.IsEmpty(job.Task().Publisher.Type) {
		cmd.Println()
		cmd.Println("To download the results, execute:")
		cmd.Printf("\t%s job get %s\n", os.Args[0], job.ID)
	}
}

func (j *JobProgressPrinter) printCancellationMessage(cmd *cobra.Command) {
	if !j.isQuiet() {
		cmd.Println()
		PrintWarning(cmd, "Progress tracking canceled. The job is still running.")
	}
}

func (j *JobProgressPrinter) printTimeoutMessage(cmd *cobra.Command) {
	if !j.isQuiet() {
		cmd.Println()
		PrintWarning(cmd, fmt.Sprintf("Progress tracking timed out after %s. The job is still running.\n", progressTimeout))
	}
}

// isQuiet returns true if the printer is in quite mode
// If we are only printing the id, set the rest of the output to "quiet",
// i.e. don't print
func (j *JobProgressPrinter) isQuiet() bool {
	return j.runtimeSettings.PrintJobIDOnly
}
