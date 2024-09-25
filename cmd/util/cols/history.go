package cols

import (
	"fmt"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

var (
	HistoryTime = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Time", WidthMax: len(time.StampMilli), WidthMaxEnforcer: output.ShortenTime},
		Value:        func(j *models.JobHistory) string { return j.Occurred().Format(time.StampMilli) },
	}
	HistoryTimeOnly = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Time", WidthMax: len(TimeOnlyMilli), WidthMaxEnforcer: text.Trim},
		Value:        func(j *models.JobHistory) string { return j.Occurred().Format(TimeOnlyMilli) },
	}
	HistoryDateTime = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Time", WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return j.Occurred().Format(time.DateTime) },
	}
	HistoryLevel = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Level", WidthMax: 15, WidthMaxEnforcer: text.WrapText},
		Value:        func(jwi *models.JobHistory) string { return jwi.Type.String() },
	}
	HistoryExecID = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Exec. ID", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.JobHistory) string {
			if j.ExecutionID == "" {
				return strings.Repeat("-", 10)
			}
			return idgen.ShortUUID(j.ExecutionID)
		},
	}
	HistoryTopic = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Topic", WidthMax: 15, WidthMaxEnforcer: text.WrapSoft},
		Value:        func(jh *models.JobHistory) string { return string(jh.Event.Topic) },
	}
	HistoryEvent = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Event", WidthMax: 90, WidthMaxEnforcer: output.WrapSoftPreserveNewlines},
		Value: func(h *models.JobHistory) string {
			res := h.Event.Message

			if h.Event.Details != nil {
				if h.Event.Details[models.DetailsKeyIsError] == "true" {
					res = output.BoldStr(output.RedStr("Error: ")) + res
				}

				// print hint in green
				if h.Event.Details[models.DetailsKeyHint] != "" {
					res +=
						"\n" + output.BoldStr(output.GreenStr("Hint: ")) + h.Event.Details[models.DetailsKeyHint]
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
