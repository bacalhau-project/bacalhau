package bacalhau

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/spf13/cobra"

	"gopkg.in/yaml.v3"
	"k8s.io/kubectl/pkg/util/i18n"
	convert "sigs.k8s.io/yaml"

	"github.com/invopop/jsonschema"
	"github.com/xeipuuv/gojsonschema"
)

var (
	validateLong = templates.LongDesc(i18n.T(`
		validate a job from a file

		JSON and YAML formats are accepted.
`))

	//nolint:lll // Documentation
	validateExample = templates.Examples(i18n.T(`
		# validate a job using the data in job.yaml
		bacalhau validate ./job.yaml
`))

	// Set Defaults (probably a better way to do this)
	OV = NewValidateOptions()

	// For the -f flag
)

type ValidateOptions struct {
	Filename string // Filename for job (can be .json or .yaml)
}

func NewValidateOptions() *ValidateOptions {
	return &ValidateOptions{
		Filename: "",
	}
}

var validateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "validate a job using a json or yaml file.",
	Long:    validateLong,
	Example: validateExample,
	Args:    cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { //nolint:unparam // incorrect that cmd is unused.

		if len(cmdArgs) == 0 {
			_ = cmd.Usage()
			return fmt.Errorf("no filename specified")
		}
		OV.Filename = cmdArgs[0]

		fileextension := filepath.Ext(OV.Filename)
		fileContent, err := os.Open(OV.Filename)

		if err != nil {
			return fmt.Errorf("could not open file '%s': %s", OV.Filename, err)
		}

		byteResult, err := io.ReadAll(fileContent)

		if err != nil {
			return err
		}

		jobSpec := &model.JobSpec{}

		if fileextension == ".json" {
			err = json.Unmarshal(byteResult, &jobSpec)
			if err != nil {
				return fmt.Errorf("error reading json file '%s': %s", OV.Filename, err)
			}
		} else if fileextension == ".yaml" || fileextension == ".yml" {
			err = yaml.Unmarshal(byteResult, &jobSpec)
			if err != nil {
				return fmt.Errorf("error reading yaml file '%s': %s", OV.Filename, err)
			}
		} else {
			return fmt.Errorf("file '%s' must be a .json or .yaml/.yml file", OV.Filename)
		}

		y2j, err := convert.YAMLToJSON(byteResult)
		if err != nil {
			return fmt.Errorf("error converting from YAML to JSON %s", err)
		}
		str := string(y2j)
		s := jsonschema.Reflect(&model.JobSpec{})
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return fmt.Errorf("error indenting %s", err)
		}
		schema := string(data)
		//nolint
		err = os.WriteFile("../../jsonschema.json", data, 0644)
		if err != nil {
			return fmt.Errorf("error writing the jsonschema JSON %s", err)
		}
		yaml, _ := convert.JSONToYAML(data)
		err = os.WriteFile("../../jsonschema.yaml", yaml, 0644)

		if err != nil {
			return fmt.Errorf("error writing the jsonschema YAML %s", err)
		}
		// fmt.Println(schema)
		if err != nil {
			return err
		}

		// println(str)
		schemaLoader := gojsonschema.NewStringLoader(schema)
		documentLoader := gojsonschema.NewStringLoader(str)

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			return err
		}

		if result.Valid() {
			fmt.Printf("The JobSpec is valid\n")
		} else {
			fmt.Printf("The JobSpec is not valid. see errors :\n")
			for _, desc := range result.Errors() {
				fmt.Printf("- %s\n", desc)
			}
		}
		return err
	},
}
