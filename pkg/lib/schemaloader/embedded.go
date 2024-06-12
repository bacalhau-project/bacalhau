package schemaloader

import (
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

// Schema is an interface that defines methods for validating data against a JSON schema.
type Schema interface {
	// ValidateFile validates the contents of a file specified by path against the schema.
	ValidateFile(path string) (*gojsonschema.Result, error)

	// ValidateReader validates the contents read from an io.Reader against the schema.
	ValidateReader(r io.Reader) (*gojsonschema.Result, error)

	// ValidateBytes validates the contents of a byte slice against the schema.
	ValidateBytes(b []byte) (*gojsonschema.Result, error)
}

// NewEmbeddedSchema returns a Schema implementation that uses an embedded filesystem to contain a JSON schema.
func NewEmbeddedSchema(fs embed.FS, root string) (Schema, error) {
	schema, err := SchemaLoader(fs, root)
	if err != nil {
		return nil, err
	}
	return &embeddedSchema{
		schema: schema,
	}, nil
}

// embeddedSchema is an implementation of the Schema interface that uses a compiled JSON schema for validation.
type embeddedSchema struct {
	schema *gojsonschema.Schema
}

// ValidateFile validates the contents of a file specified by path against the schema.
func (e *embeddedSchema) ValidateFile(path string) (*gojsonschema.Result, error) {
	// Ensure we are given a json or yaml file only.
	fileExt := filepath.Ext(path)
	if !(fileExt == ".json" || fileExt == ".yaml" || fileExt == ".yml") {
		return nil, fmt.Errorf("file extension (%s) not supported. The file must end in either .yaml, .yml or .json", fileExt)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file (%s): %w", path, err)
	}
	defer file.Close()
	return e.ValidateReader(file)
}

// ValidateReader validates the contents read from an io.Reader against the schema.
func (e *embeddedSchema) ValidateReader(r io.Reader) (*gojsonschema.Result, error) {
	inputData, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return e.ValidateBytes(inputData)
}

// ValidateBytes validates the contents of a byte slice against the schema.
func (e *embeddedSchema) ValidateBytes(b []byte) (*gojsonschema.Result, error) {
	// Convert the document to JSON if it's in YAML format
	document, err := yaml.YAMLToJSON(b)
	if err != nil {
		return nil, fmt.Errorf("converting yaml to json: %w", err)
	}

	documentLoader := gojsonschema.NewStringLoader(string(document))

	return e.schema.Validate(documentLoader)
}

// newEmbedFS creates an http.FileSystem from the embedded filesystem.
func newEmbedFS(fs embed.FS) http.FileSystem {
	return http.FS(fs)
}

// SchemaLoader loads and compiles the main schema along with its dependencies from the embedded filesystem.
func SchemaLoader(fs embed.FS, mainSchemaFile string) (*gojsonschema.Schema, error) {
	embedFS := newEmbedFS(fs)

	// Load and compile the main schema using NewReferenceLoaderFileSystem
	mainSchemaLoader := gojsonschema.NewReferenceLoaderFileSystem("file:///"+mainSchemaFile, embedFS)

	// Compile the schema
	schema, err := gojsonschema.NewSchema(mainSchemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile main schema: %w", err)
	}

	return schema, nil
}
