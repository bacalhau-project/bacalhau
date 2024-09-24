// Package printer provides functionality for printing job events and progress.
// It offers different types of printers to handle various output formats and requirements.
package printer

import (
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/cols"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// eventPrinter defines the interface for different types of event printers.
type eventPrinter interface {
	// printEvent prints a single job history event.
	printEvent(history *models.JobHistory) error
	// close performs any necessary cleanup operations.
	close() error
}

// quitePrinter is an eventPrinter that discards all output.
type quitePrinter struct {
}

// newQuitePrinter creates a new quitePrinter and sets the command output to discard.
func newQuitePrinter(cmd *cobra.Command) *quitePrinter {
	return &quitePrinter{}
}

// printEvent implements eventPrinter interface but does nothing for quitePrinter.
func (quitePrinter) printEvent(*models.JobHistory) error { return nil }

// close implements eventPrinter interface but does nothing for quitePrinter.
func (quitePrinter) close() error { return nil }

// sequentialEventPrinter prints job events sequentially as they occur.
type sequentialEventPrinter struct {
	cmd                 *cobra.Command
	columns             []output.TableColumn[*models.JobHistory]
	lineCount           int
	spinner             *FishSpinner
	seenExecutionErrors bool
}

// newSequentialEventPrinter creates a new sequentialEventPrinter.
func newSequentialEventPrinter(cmd *cobra.Command) *sequentialEventPrinter {
	spinner := NewFishSpinner(cmd.OutOrStdout())
	spinner.Start()

	eventsCols := []output.TableColumn[*models.JobHistory]{
		cols.HistoryTimeOnly,
		cols.HistoryExecID,
		cols.HistoryTopic,
		cols.HistoryEvent,
	}

	// since we print each row as a separate table, we need to set the
	// min width of the columns to avoid inconsistent column widths.
	// will use the same as max width for now, except for the last column.
	for i := 0; i < len(eventsCols)-1; i++ {
		eventsCols[i].WidthMin = eventsCols[i].WidthMax
	}

	return &sequentialEventPrinter{
		cmd:     cmd,
		columns: eventsCols,
		spinner: spinner,
	}
}

// filterEvent determines whether an event should be printed.
// It filters out events that can be too noisy.
func (p *sequentialEventPrinter) filterEvent(event *models.JobHistory) bool {
	// Always print the first event indicating job submission
	if p.lineCount == 0 {
		return true
	}

	// Check for execution level errors
	if event.IsExecutionLevel() && event.Event.HasError() {
		p.seenExecutionErrors = true
		return true
	}

	// Print job level errors only if we haven't seen execution level errors
	if !event.IsExecutionLevel() && event.Event.HasError() && !p.seenExecutionErrors {
		return true
	}

	// Print all non-error execution level events
	if event.IsExecutionLevel() {
		return true
	}

	return false
}

// printEvent prints a single job history event if it passes the filter.
func (p *sequentialEventPrinter) printEvent(event *models.JobHistory) error {
	if !p.filterEvent(event) {
		return nil
	}

	options := output.OutputOptions{Format: output.TableFormat, NoStyle: true, HideHeader: true}
	if p.lineCount == 0 {
		options.HideHeader = false
	}

	p.spinner.Pause() // Pause the spinner while printing the event
	p.spinner.Clear() // Clear the spinner

	err := output.Output(p.cmd, p.columns, options, []*models.JobHistory{event})
	if err != nil {
		return err
	}

	p.spinner.Resume() // Resume the spinner
	p.lineCount++

	return nil
}

// close clears the loading message when closing the printer.
func (p *sequentialEventPrinter) close() error {
	p.spinner.Stop()
	p.spinner.Clear()
	return nil
}

// groupedEventPrinter is an experimental printer that groups events by execution.
// UNSTABLE: This printer is still in development and may change in future versions.
type groupedEventPrinter struct {
	cmd         *cobra.Command
	columns     []output.TableColumn[*models.JobHistory]
	existingOut io.Writer
	jobEvent    *models.JobHistory
	executions  []*executionGroup
}

type executionGroup struct {
	ExecutionID   string
	LatestEvent   *models.JobHistory
	DiscoveryTime time.Time
}

// newGroupedEventPrinter creates a new groupedEventPrinter.
func newGroupedEventPrinter(cmd *cobra.Command) *groupedEventPrinter {
	existingOut := cmd.OutOrStdout()
	cmd.SetOut(util.NewLiveTableWriter())

	eventsCols := []output.TableColumn[*models.JobHistory]{
		cols.HistoryTimeOnly,
		cols.HistoryExecID,
		cols.HistoryTopic,
		cols.HistoryEvent,
	}

	return &groupedEventPrinter{
		cmd:         cmd,
		existingOut: existingOut,
		columns:     eventsCols,
		executions:  make([]*executionGroup, 0),
	}
}

// printEvent adds or updates an event and renders the table.
func (p *groupedEventPrinter) printEvent(event *models.JobHistory) error {
	if event.ExecutionID == "" {
		p.jobEvent = event
	} else {
		p.updateOrAddExecution(event)
	}
	return p.renderTable()
}

// updateOrAddExecution updates an existing execution or adds a new one.
func (p *groupedEventPrinter) updateOrAddExecution(event *models.JobHistory) {
	for _, group := range p.executions {
		if group.ExecutionID == event.ExecutionID {
			group.LatestEvent = event
			return
		}
	}

	// If execution not found, add a new one
	newGroup := &executionGroup{
		ExecutionID:   event.ExecutionID,
		LatestEvent:   event,
		DiscoveryTime: time.Now(),
	}
	p.executions = append(p.executions, newGroup)
}

// renderTable outputs all events as a table with stable execution order.
func (p *groupedEventPrinter) renderTable() error {
	var entries []*models.JobHistory

	// Add execution events in the order they were discovered
	for _, group := range p.executions {
		entries = append(entries, group.LatestEvent)
	}

	// Add job event last
	if p.jobEvent != nil {
		entries = append(entries, p.jobEvent)
	}

	options := output.OutputOptions{Format: output.TableFormat, NoStyle: true}
	return output.Output(p.cmd, p.columns, options, entries)
}

// close resets the command's output and flushes the live writer.
func (p *groupedEventPrinter) close() error {
	p.cmd.SetOut(p.existingOut)
	return nil
}
