package translators

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type DuckDBTranslator struct{}

func (d *DuckDBTranslator) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (d *DuckDBTranslator) Translate(original *models.Task) (*models.Task, error) {
	builder := original.
		ToBuilder().
		Engine(d.dockerEngine(original.Engine))

	return builder.BuildOrDie(), nil
}

func (d *DuckDBTranslator) dockerEngine(origin *models.SpecConfig) *models.SpecConfig {
	// It'd be nice to use pkg/executor/docker/types/EngineSpec here, but it
	// would mean adding a dependency on yet another package.
	cmd := origin.Params["Command"].(string)
	args := origin.Params["Arguments"].([]string)

	params := []string{}

	params = append(params, cmd)
	params = append(params, args...)

	spec := models.NewSpecConfig(models.EngineDocker)
	spec.Params["Image"] = "bacalhauproject/exec-duckdb:0.1"
	spec.Params["Entrypoint"] = []string{}
	spec.Params["Parameters"] = params
	spec.Params["EnvironmentVariables"] = []string{}
	spec.Params["WorkingDirectory"] = ""

	return spec
}
