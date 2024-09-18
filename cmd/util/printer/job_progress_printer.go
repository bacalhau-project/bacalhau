package printer

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const PrintoutCanceledButRunningNormally string = "printout canceled but running normally"

type JobProgressPrinter struct {
	client          clientv2.API
	runtimeSettings *cliflags.RunTimeSettings
}

type jobProgressEvent struct {
	jobID       string
	occurred    time.Time
	executionID string
	event       models.Event
}

var (
	jobProgressEventTimeCol = output.TableColumn[*jobProgressEvent]{
		ColumnConfig: table.ColumnConfig{Name: "Time", WidthMax: len(time.StampMilli), WidthMaxEnforcer: output.ShortenTime},
		Value:        func(j *jobProgressEvent) string { return j.occurred.Format(time.StampMilli) },
	}

	jobProgressEventExecIDCol = output.TableColumn[*jobProgressEvent]{
		ColumnConfig: table.ColumnConfig{Name: "Exec. ID", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *jobProgressEvent) string { return idgen.ShortUUID(j.executionID) },
	}

	jobProgressEventTopicCol = output.TableColumn[*jobProgressEvent]{
		ColumnConfig: table.ColumnConfig{Name: "Topic", WidthMax: 20, WidthMaxEnforcer: text.WrapSoft},
		Value: func(j *jobProgressEvent) string {
			return string(j.event.Topic)
		},
	}

	jobProgressEventEventCol = output.TableColumn[*jobProgressEvent]{
		ColumnConfig: table.ColumnConfig{Name: "Event", WidthMax: 60, WidthMaxEnforcer: text.WrapText},
		Value: func(j *jobProgressEvent) string {
			res := j.event.Message

			if j.event.Details != nil {
				// if is error, then the event is in red
				if j.event.Details[models.DetailsKeyIsError] == "true" {
					res = output.RedStr(res)
				}

				// print hint in green
				if j.event.Details[models.DetailsKeyHint] != "" {
					res += "\n" + fmt.Sprintf(
						"%s %s", output.BoldStr(output.GreenStr("* Hint:")), j.event.Details[models.DetailsKeyHint])
				}

				// print all other details in debug mode
				if zerolog.GlobalLevel() <= zerolog.DebugLevel {
					for k, v := range j.event.Details {
						// don't print hint and error since they are already represented
						if k == models.DetailsKeyHint || k == models.DetailsKeyIsError {
							continue
						}
						res += "\n" + fmt.Sprintf("* %s %s", output.BoldStr(k+":"), v)
					}
				}
			}
			return res
		},
	}
)

var jobProgressEventCols = []output.TableColumn[*jobProgressEvent]{
	jobProgressEventTimeCol,
	jobProgressEventExecIDCol,
	jobProgressEventTopicCol,
	jobProgressEventEventCol,
}

func NewJobProgressPrinter(client clientv2.API, runtimeSettings *cliflags.RunTimeSettings) *JobProgressPrinter {
	return &JobProgressPrinter{
		client:          client,
		runtimeSettings: runtimeSettings,
	}
}

// PrintJobProgress displays the job progress depending upon on cli runtime
// settings
func (j *JobProgressPrinter) PrintJobProgress(ctx context.Context, job *models.Job, jobID string, cmd *cobra.Command) error {
	// If we are in `--wait=false` print the jobID and then exit.
	// All the code after this point is to show the progress of the job.
	if !j.runtimeSettings.WaitForJobToFinish {
		cmd.Print(jobID + "\n")
		return nil
	}

	// If we are n `id-only` mode - print the id
	if j.runtimeSettings.PrintJobIDOnly {
		cmd.Print(jobID + "\n")
	}

	// Follow Logs
	if j.runtimeSettings.Follow {
		return j.followLogs(jobID, cmd)
	}

	// If we are only printing the id, set the rest of the output to "quiet",
	// i.e. don't print
	quiet := j.runtimeSettings.PrintJobIDOnly

	jobErr := j.followProgress(ctx, jobID, cmd, quiet)
	if jobErr != nil {
		if jobErr.Error() == PrintoutCanceledButRunningNormally {
			return nil
		}

		history, err := j.client.Jobs().History(ctx, &apimodels.ListJobHistoryRequest{
			JobID:     jobID,
			EventType: "execution",
		})
		if err != nil {
			return fmt.Errorf("failed getting job history: %w", err)
		}

		historySummary := SummariseHistoryEvents(history.Items)
		if len(historySummary) > 0 {
			for _, event := range historySummary {
				PrintEvent(cmd, event)
			}
		} else {
			PrintError(cmd, jobErr)
		}
	}

	if j.runtimeSettings.PrintNodeDetails || jobErr != nil {
		executions, err := j.client.Jobs().Executions(ctx, &apimodels.ListJobExecutionsRequest{
			JobID: jobID,
		})
		if err != nil {
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

		// only print help for downloading the job if it contained a publisher.
		if !lo.IsEmpty(job.Task().Publisher.Type) {
			cmd.Println()
			cmd.Println("To download the results, execute:")
			cmd.Println("\t" + os.Args[0] + " job get " + jobID)
		}
	}

	return nil
}

func (j *JobProgressPrinter) followLogs(jobID string, cmd *cobra.Command) error {
	cmd.Printf("Job successfully submitted. Job ID: %s\n", jobID)
	cmd.Printf("Waiting for logs... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")

	// Wait until the job has actually been accepted and started, otherwise this will fail waiting for
	// the execution to appear.
	for i := 0; i < 10; i++ {
		resp, err := j.client.Jobs().Get(cmd.Context(), &apimodels.GetJobRequest{
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

	return util.Logs(cmd, j.client, util.LogOptions{
		JobID:  jobID,
		Follow: true,
	})
}

//nolint:gocyclo,funlen
func (j *JobProgressPrinter) followProgress(ctx context.Context, jobID string, cmd *cobra.Command, quiet bool) error {
	getMoreInfoString := fmt.Sprintf(`
To get more information at any time, run:
   bacalhau job describe %s`, jobID)

	cancelString := fmt.Sprintf(`
To cancel the job, run:
   bacalhau job stop %s`, jobID)

	if !quiet {
		cmd.Printf("Job successfully submitted. Job ID: %s\n", jobID)
		cmd.Print("Checking job status... (Enter Ctrl+C to exit at any time, your job will continue running):\n\n")
	}

	cmdShuttingDown := false
	var returnError error = nil

	// Capture Ctrl + C if the user wants to finish the job early
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, util.ShutdownSignals...)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()

	liveTableWriter := util.NewLiveTableWriter()
	if quiet {
		cmd.SetOut(io.Discard)
	} else {
		cmd.SetOut(liveTableWriter)
	}

	// goroutine for handling SIGINT from the signal channel, or context completion messages.
	go func() {
		for {
			select {
			case s := <-signalChan:
				log.Ctx(ctx).Debug().Msgf("Captured %v. Exiting...", s)
				if s == os.Interrupt {
					cmdShuttingDown = true
					cmd.SetOut(os.Stdout)

					if !quiet {
						cmd.Println("\n\n\rPrintout canceled (the job is still running).")
						cmd.Println(getMoreInfoString)
						cmd.Println(cancelString)
					}
					returnError = fmt.Errorf("%s", PrintoutCanceledButRunningNormally)
				} else {
					cmd.Println("Unexpected signal received. Exiting.")
				}
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	var currentJobState models.JobStateType
	nextToken := ""
	jobProgressEvents := make(map[string]*jobProgressEvent)
	tableOptions := output.OutputOptions{
		Format:  output.TableFormat,
		NoStyle: true,
	}

	timeFilter := time.Now().Unix()

	for !cmdShuttingDown {
		// With new history events, we no longer support job state. Thus we need
		// to fetch the job to get the job state, and determine if the job has completed or not.
		resp, err := j.client.Jobs().Get(ctx, &apimodels.GetJobRequest{
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

		currentJobState = resp.Job.State.StateType

		// Have we been cancel(l)ed?
		if condition := ctx.Err(); condition != nil {
			break
		}

		// We are separating out 2 concerns here
		// 1. If the Job is in terminal state, we just fetch all the remaining history events, as
		//    the command will shut down after it.
		// 2. If the Job is not in terminal state, we fetch single events, this is better while observing
		//    history as only one line gets updated. Hence a good user experience.
		var jobHistoryRequest apimodels.ListJobHistoryRequest
		if currentJobState.IsTerminal() {
			jobHistoryRequest = apimodels.ListJobHistoryRequest{
				JobID:     jobID,
				EventType: "all",
				Since:     timeFilter,
			}
			if currentJobState != models.JobStateTypeCompleted {
				returnError = errors.New("job failed")
			}
			cmdShuttingDown = true
		} else {
			jobHistoryRequest = apimodels.ListJobHistoryRequest{
				JobID:     jobID,
				EventType: "all",
				Since:     timeFilter,
				BaseListRequest: apimodels.BaseListRequest{
					NextToken: nextToken,
					Limit:     1,
				},
			}
		}

		jobHistoryResponse, _ := j.client.Jobs().History(ctx, &jobHistoryRequest)
		// We need to make sure if the items returned are zero, then we update timeFilter to current time
		// As or else the timeFilter would be an older history event, we would keep on getting already
		// displayed history events.
		if len(jobHistoryResponse.Items) == 0 {
			timeFilter = time.Now().Unix()
		}

		nextToken = jobHistoryResponse.NextToken

		for _, history := range jobHistoryResponse.Items {
			timeFilter = history.Time.Unix()

			// We group based at 2 levels
			// 1. Per Execution
			// 2. Job Level State Changes
			var eventID string
			if history.Type == models.JobHistoryTypeExecutionLevel {
				eventID = history.ExecutionID
			} else {
				eventID = history.Type.String()
			}

			jobProgressEvents[eventID] = &jobProgressEvent{
				jobID:       jobID,
				occurred:    history.Occurred(),
				executionID: history.ExecutionID,
				event:       history.Event,
			}
		}

		// Displays Job Progress Output in Table Format
		entries := lo.Entries(jobProgressEvents)

		// We do custom sorting mainly because, there is a chance that
		// both execution level and job level states have same timestamp. In that scenarion,
		// we need to make sure the order is still determinant. The table sorted does not
		// support this hence we do custom sorting. rd
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Value.occurred.Before(entries[j].Value.occurred) {
				return true
			} else if entries[i].Value.occurred.After(entries[j].Value.occurred) {
				return false
			} else {
				return entries[i].Value.event.Topic < entries[j].Value.event.Topic
			}
		})
		if err := output.Output(cmd, jobProgressEventCols, tableOptions,
			lo.Map(entries, func(item lo.Entry[string, *jobProgressEvent], index int) *jobProgressEvent {
				return item.Value
			})); err != nil {
			return fmt.Errorf("failed to print job progress: %w", err)
		}

		// Have we been cancel(l)ed?
		if condition := ctx.Err(); condition != nil {
			break
		}

		time.Sleep(time.Millisecond * 500) //nolint:mnd // 500ms sleep
	}

	// This is needed as while printing progress, we delegate printing job progress table
	// to Live Table Writer. We eventually close it. Now since it is closed, we want the
	// command to redirect the output to Stdout again.
	cmd.SetOut(os.Stdout)
	return returnError
}
