package printer

import (
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/rs/zerolog"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

var (
	eventsCols = []output.TableColumn[*models.JobHistory]{
		eventTimeCol,
		eventExecIDCol,
		eventTopicCol,
		eventEventCol,
	}
)

var (
	eventTimeCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Time",
			WidthMin: 20, WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.JobHistory) string { return j.Time.Format(time.DateTime) },
	}

	eventExecIDCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Exec. ID", WidthMin: 12, WidthMax: 12, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.JobHistory) string {
			if j.ExecutionID == "" {
				return "/"
			}
			return idgen.ShortUUID(j.ExecutionID)
		},
	}

	eventTopicCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{Name: "Topic", WidthMin: 18, WidthMax: 18, WidthMaxEnforcer: text.WrapSoft},
		Value: func(j *models.JobHistory) string {
			return string(j.Event.Topic)
		},
	}

	eventEventCol = output.TableColumn[*models.JobHistory]{
		ColumnConfig: table.ColumnConfig{
			Name: "Event", WidthMin: 30, WidthMax: 90, WidthMaxEnforcer: output.WrapSoftPreserveNewlines},
		Value: func(j *models.JobHistory) string {
			res := j.Event.Message

			if j.Event.Details != nil {
				// if is error, then the event is in red
				if j.Event.Details[models.DetailsKeyIsError] == "true" {
					res = output.BoldStr(output.RedStr("Error: ")) + res
				}

				// print hint in green
				if j.Event.Details[models.DetailsKeyHint] != "" {
					res +=
						"\n" + output.BoldStr(output.GreenStr("Hint: ")) + j.Event.Details[models.DetailsKeyHint]
				}

				// print all other details in debug mode
				if zerolog.GlobalLevel() <= zerolog.DebugLevel {
					for k, v := range j.Event.Details {
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
