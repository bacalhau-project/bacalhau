package bacalhau

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"

	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/xeipuuv/gojsonschema"

	"github.com/tidwall/sjson"
)

var (
	validateLong = templates.LongDesc(i18n.T(`
		Validate a job from a file

		JSON and YAML formats are accepted.
`))

	//nolint:lll // Documentation
	validateExample = templates.Examples(i18n.T(`
		# Validate a job using the data in job.yaml
		bacalhau validate ./job.yaml

		# Validate a job using stdin
		cat job.yaml | bacalhau validate

		# Output the jsonschema for a bacalhau job
		bacalhau validate --output-schema
`))

	// Set Defaults (probably a better way to do this)
	OV = NewValidateOptions()
)

type ValidateOptions struct {
	Filename        string // Filename for job (can be .json or .yaml)
	OutputFormat    string // Output format (json or yaml)
	OutputSchema    bool   // Output the schema to stdout
	OutputDirectory string // Output directory for the job
}

func NewValidateOptions() *ValidateOptions {
	return &ValidateOptions{
		Filename:        "",
		OutputFormat:    "yaml",
		OutputSchema:    false,
		OutputDirectory: "",
	}
}

func init() { //nolint:gochecknoinits // Using init with Cobra Command is ideomatic
	validateCmd.PersistentFlags().BoolVar(
		&OV.OutputSchema, "output-schema", OV.OutputSchema,
		`Output the JSON schema for a Job to stdout then exit`,
	)
}

var validateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "validate a job using a json or yaml file.",
	Long:    validateLong,
	Example: validateExample,
	Args:    cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { //nolint:unparam // incorrect that cmd is unused.
		j := &model.Job{}
		jsonSchemaData, err := GenerateJobJSONSchema()
		if err != nil {
			return err
		}

		if OV.OutputSchema {
			//nolint
			cmd.Printf("%s", jsonSchemaData)
			return nil
		}

		if len(cmdArgs) == 0 {
			_ = cmd.Usage()
			Fatal("You must specify a filename or provide the content to be validated via stdin.", 1)
		}

		OV.Filename = cmdArgs[0]
		var byteResult []byte

		if OV.Filename == "" {
			// Read from stdin
			byteResult, err = io.ReadAll(cmd.InOrStdin())
			if err != nil {
				Fatal(fmt.Sprintf("Error reading from stdin: %s", err), 1)
			}
			if byteResult == nil {
				// Can you ever get here?
				Fatal("No filename provided.", 1)
			}
		} else {
			var file *os.File
			fileextension := filepath.Ext(OV.Filename)
			file, err = os.Open(OV.Filename)

			if err != nil {
				Fatal(fmt.Sprintf("Error opening file (%s): %s", OV.Filename, err), 1)
			}

			byteResult, err = io.ReadAll(file)

			if err != nil {
				return err
			}

			if fileextension == ".json" || fileextension == ".yaml" || fileextension == ".yml" {
				// Yaml can parse json
				err = model.YAMLUnmarshalWithMax(byteResult, &j)
				if err != nil {
					Fatal(fmt.Sprintf("Error unmarshaling yaml from file (%s): %s", OV.Filename, err), 1)
				}
			} else {
				Fatal(fmt.Sprintf("File extension (%s) not supported. The file must end in either .yaml, .yml or .json.", fileextension), 1)
			}

		}
		// Convert the schema to JSON - this is required for the gojsonschema library
		// Noop if you pass JSON through
		fileContentsAsJSONBytes, err := yaml.YAMLToJSON(byteResult)
		if err != nil {
			Fatal(fmt.Sprintf("Error converting yaml to json: %s", err), 1)
		}

		// println(str)
		schemaLoader := gojsonschema.NewStringLoader(string(jsonSchemaData))
		documentLoader := gojsonschema.NewStringLoader(string(fileContentsAsJSONBytes))

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			Fatal(fmt.Sprintf("Error validating json: %s", err), 1)
		}

		if result.Valid() {
			cmd.Println("The Job is valid")
		} else {
			msg := "The Job is not valid. See errors:\n"
			for _, desc := range result.Errors() {
				msg += fmt.Sprintf("- %s\n", desc)
			}
			Fatal(msg, 1)
		}
		return nil
	},
}

func GenerateJobJSONSchema() ([]byte, error) {
	s := jsonschema.Reflect(&model.Job{})
	// Find key in a json document in Golang
	// https://stackoverflow.com/questions/52953282/how-to-find-a-key-in-a-json-document

	jsonSchemaData, err := model.JSONMarshalIndentWithMax(s, 2)
	if err != nil {
		return nil, fmt.Errorf("error indenting %s", err)
	}

	// JSON String
	jsonString := string(jsonSchemaData)

	enumTypes := []struct {
		Name  string
		Path  string
		Enums []string
	}{
		{Name: "Engine",
			Path:  "$defs.Spec.properties.Engine",
			Enums: model.EngineNames()},
		{Name: "Verifier",
			Path:  "$defs.Spec.properties.Verifier",
			Enums: model.VerifierNames()},
		{Name: "Publisher",
			Path:  "$defs.Spec.properties.Publisher",
			Enums: model.PublisherNames()},
		{Name: "StorageSource",
			Path:  "$defs.StorageSpec.properties.StorageSource",
			Enums: model.StorageSourceNames()},
	}
	for _, enumType := range enumTypes {
		// Use sjson to find the enum type path in the JSON
		jsonString, _ = sjson.Set(jsonString, enumType.Path+".type", "string")

		jsonString, _ = sjson.Set(jsonString, enumType.Path+".enum", enumType.Enums)
	}

	return []byte(jsonString), nil
}
