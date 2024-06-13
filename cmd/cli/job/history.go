package job

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	historyShort = `List history events for a job by id.`

	historyLong = templates.LongDesc(i18n.T(`
		List job history events for a job by id.
`))

	historyExample = templates.Examples(i18n.T(`
		# All events for a given job.
		bacalhau job history e3f8c209-d683-4a41-b840-f09b88d087b9

		# Job level events
		bacalhau job history --type job e3f8c209

		# Execution level events
		bacalhau job history --type execution e3f8c209
`))
)

// HistoryOptions is a struct to support node command
type HistoryOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	EventType   string
	ExecutionID string
	NodeID      string
}

// NewHistoryOptions returns initialized Options
func NewHistoryOptions() *HistoryOptions {
	return &HistoryOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
		EventType:     "all",
	}
}

func NewHistoryCmd() *cobra.Command {
	o := NewHistoryOptions()
	nodeCmd := &cobra.Command{
		Use:     "history [id]",
		Short:   historyShort,
		Long:    historyLong,
		Example: historyExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.run(cmd, args, api)
		},
	}

	nodeCmd.Flags().StringVar(&o.EventType, "event-type", o.EventType,
		"The type of history events to return. One of: all, job, execution")
	nodeCmd.Flags().StringVar(&o.ExecutionID, "execution-id", o.ExecutionID,
		"The execution id to filter by.")
	nodeCmd.Flags().StringVar(&o.NodeID, "node-id", o.NodeID,
		"The node id to filter by.")
	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return nodeCmd
}

var (
	historyTimeCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Time", WidthMax: len(time.StampMilli), WidthMaxEnforcer: output.ShortenTime},
		Value:        func(j *models.JobHistory) string { return j.Occurred().Format(time.StampMilli) },
	}
	historyLevelCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Level", WidthMax: 15, WidthMaxEnforcer: text.WrapText},
		Value:        func(jwi *models.JobHistory) string { return jwi.Type.String() },
	}
	historyRevisionCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Rev.", WidthMax: 4, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return strconv.FormatUint(j.NewRevision, 10) },
	}
	historyExecIDCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Exec. ID", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return idgen.ShortUUID(j.ExecutionID) },
	}
	historyNodeIDCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Node ID", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return idgen.ShortNodeID(j.NodeID) },
	}
	historyStateCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "State", WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.JobHistory) string {
			if j.Type == models.JobHistoryTypeJobLevel {
				return j.JobState.New.String()
			}
			return j.ExecutionState.New.String()
		},
	}
	historyTopicCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Topic", WidthMax: 15, WidthMaxEnforcer: text.WrapSoft},
		Value:        func(jh *models.JobHistory) string { return string(jh.Event.Topic) },
	}
	historyEventCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Event", WidthMax: 60, WidthMaxEnforcer: text.WrapText},
		Value: func(h *models.JobHistory) string {
			res := h.Event.Message

			if h.Event.Details != nil {
				// if is error, then the event is in red
				if h.Event.Details[models.DetailsKeyIsError] == "true" {
					res = output.RedStr(res)
				}

				// print hint in green
				if h.Event.Details[models.DetailsKeyHint] != "" {
					res += "\n" + fmt.Sprintf(
						"%s %s", output.BoldStr(output.GreenStr("* Hint:")), h.Event.Details[models.DetailsKeyHint])
				}

				// print all other details in debug mode
				if zerolog.GlobalLevel() <= zerolog.DebugLevel {
					for k, v := range h.Event.Details {
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

var historyColumns = []output.TableColumn[*models.JobHistory]{
	historyTimeCol,
	historyLevelCol,
	historyRevisionCol,
	historyExecIDCol,
	historyNodeIDCol,
	historyStateCol,
	historyTopicCol,
	historyEventCol,
}

func (o *HistoryOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	jobID := args[0]
	response, err := api.Jobs().History(ctx, &apimodels.ListJobHistoryRequest{
		JobID:       jobID,
		EventType:   o.EventType,
		ExecutionID: o.ExecutionID,
		NodeID:      o.NodeID,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.Limit,
			NextToken: o.NextToken,
			OrderBy:   o.OrderBy,
			Reverse:   o.Reverse,
		},
	})
	if err != nil {
		return err
	}

	if err = output.Output(cmd, historyColumns, o.OutputOptions, response.History); err != nil {
		return fmt.Errorf("failed to output: %w", err)
	}

	return nil
}
