package dashboard

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/dashboard/api/pkg/model"
	bacalhau_model_v1alpha1 "github.com/filecoin-project/bacalhau/pkg/model/v1alpha1"
	bacalhau_model_v1beta1 "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/spf13/cobra"
)

const TotalLogLines = 3226637

type importOptionsType struct {
	filename string
}

func setupImportOptions(cmd *cobra.Command, opts *importOptionsType) {
	cmd.PersistentFlags().StringVar(
		&opts.filename, "filename", opts.filename,
		`The filename to import logs from`,
	)
}

func newImportOptions() importOptionsType {
	return importOptionsType{
		filename: "",
	}
}

func newImportCommand() *cobra.Command {
	modelOptions := newModelOptions()
	importOptions := newImportOptions()
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import JSON log events",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return importLogs(cmd, modelOptions, importOptions)
		},
	}
	setupModelOptions(importCmd, &modelOptions)
	setupImportOptions(importCmd, &importOptions)
	return importCmd
}

type LogLineAlpha struct {
	Type  string
	Event bacalhau_model_v1alpha1.JobEvent
}

type LogLineBeta struct {
	Type  string
	Event bacalhau_model_v1beta1.JobEvent
}

func importLogs(cmd *cobra.Command, modelOptions model.ModelOptions, opts importOptionsType) error {
	model, err := model.NewModelAPI(modelOptions)
	if err != nil {
		return err
	}

	if opts.filename == "" {
		return fmt.Errorf("please specify a filename")
	}

	_, err = os.Stat(opts.filename)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("filename does not exist: %s", opts.filename)
	}

	file, err := os.Open(opts.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	counter := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		counter++
		var event bacalhau_model_v1beta1.JobEvent
		text := scanner.Text()
		if strings.Contains(text, `"APIVersion":"V1beta1"`) {
			var line LogLineBeta
			err = json.Unmarshal([]byte(text), &line)
			if err != nil {
				return err
			}
			if line.Type != "model.JobEvent" {
				return fmt.Errorf("expected JobEvent, got %s", line.Type)
			}
			event = line.Event
		} else {
			var line LogLineAlpha
			err = json.Unmarshal([]byte(text), &line)
			if err != nil {
				return err
			}
			if line.Type != "model.JobEvent" {
				return fmt.Errorf("expected JobEvent, got %s", line.Type)
			}
			event = bacalhau_model_v1beta1.ConvertV1alpha1JobEvent(line.Event)
		}
		fmt.Printf("%d / %d event %s %s\n", counter, TotalLogLines, event.JobID, event.EventName.String())
		err = model.AddEvent(event)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
