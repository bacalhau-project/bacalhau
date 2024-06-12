package job

import (
	"embed"

	"github.com/bacalhau-project/bacalhau/pkg/lib/schemaloader"
)

//go:embed schemas/**
var jsonSchemas embed.FS

const mainSchemaFile = "schemas/job-schema.json"

func Schema() (schemaloader.Schema, error) {
	return schemaloader.NewEmbeddedSchema(jsonSchemas, mainSchemaFile)
}
