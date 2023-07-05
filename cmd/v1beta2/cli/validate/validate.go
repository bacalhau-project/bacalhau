package validate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/v1beta2/util"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"

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

func NewCmd() *cobra.Command {
	OV := NewValidateOptions()

	validateCmd := &cobra.Command{
		Use:     "validate",
		Short:   "validate a job using a json or yaml file.",
		Long:    validateLong,
		Example: validateExample,
		Args:    cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, cmdArgs []string) { //nolint:unparam // incorrect that cmd is unused.
			if err := validate(cmd, cmdArgs, OV); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	validateCmd.PersistentFlags().BoolVar(
		&OV.OutputSchema, "output-schema", OV.OutputSchema,
		`Output the JSON schema for a Job to stdout then exit`,
	)

	return validateCmd
}

func validate(cmd *cobra.Command, cmdArgs []string, OV *ValidateOptions) error {
	j := &v1beta2.Job{}
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
		return fmt.Errorf("you must specify a filename or provide the content to be validated via stdin")
	}

	OV.Filename = cmdArgs[0]
	var byteResult []byte

	if OV.Filename == "" {
		// Read from stdin
		byteResult, err = io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("error reading from stdin: %w", err)
		}
		if byteResult == nil {
			// Can you ever get here?
			return fmt.Errorf("no filename provided")
		}
	} else {
		fileextension := filepath.Ext(OV.Filename)
		file, err := os.Open(OV.Filename)
		if err != nil {
			return fmt.Errorf("error opening file (%s): %w", OV.Filename, err)
		}

		byteResult, err = io.ReadAll(file)
		if err != nil {
			return err
		}

		if fileextension == ".json" || fileextension == ".yaml" || fileextension == ".yml" {
			// Yaml can parse json
			err = v1beta2.YAMLUnmarshalWithMax(byteResult, &j)
			if err != nil {
				return fmt.Errorf("error unmarshaling yaml from file (%s): %w", OV.Filename, err)
			}
		} else {
			return fmt.Errorf("file extension (%s) not supported. The file must end in either .yaml, .yml or .json", fileextension)
		}
	}

	// Convert the schema to JSON - this is required for the gojsonschema library
	// Noop if you pass JSON through
	fileContentsAsJSONBytes, err := yaml.YAMLToJSON(byteResult)
	if err != nil {
		return fmt.Errorf("error converting yaml to json: %w", err)
	}

	// println(str)
	schemaLoader := gojsonschema.NewStringLoader(string(jsonSchemaData))
	documentLoader := gojsonschema.NewStringLoader(string(fileContentsAsJSONBytes))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("error validating json: %w", err)
	}

	if result.Valid() {
		cmd.Println("The Job is valid")
	} else {
		msg := "The Job is not valid. See errors:\n"
		for _, desc := range result.Errors() {
			msg += fmt.Sprintf("- %s\n", desc)
		}
		return fmt.Errorf(msg)
	}
	return nil
}

func GenerateJobJSONSchema() ([]byte, error) {
	s := jsonschema.Reflect(&v1beta2.Job{})
	// Find key in a json document in Golang
	// https://stackoverflow.com/questions/52953282/how-to-find-a-key-in-a-json-document

	jsonSchemaData, err := v1beta2.JSONMarshalIndentWithMax(s, 2)
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
			Enums: v1beta2.EngineNames()},
		{Name: "Verifier",
			Path:  "$defs.Spec.properties.Verifier",
			Enums: v1beta2.VerifierNames()},
		{Name: "Publisher",
			Path:  "$defs.Spec.properties.Publisher",
			Enums: v1beta2.PublisherNames()},
		{Name: "StorageSource",
			Path:  "$defs.StorageSpec.properties.StorageSource",
			Enums: v1beta2.StorageSourceNames()},
	}
	for _, enumType := range enumTypes {
		// Use sjson to find the enum type path in the JSON
		jsonString, _ = sjson.Set(jsonString, enumType.Path+".type", "string")

		jsonString, _ = sjson.Set(jsonString, enumType.Path+".enum", enumType.Enums)
	}

	return []byte(jsonString), nil
}
