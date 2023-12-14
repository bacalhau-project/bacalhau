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
	dkrSpec, err := d.dockerEngine(original.Engine)
	if err != nil {
		return nil, err
	}

	builder := original.
		ToBuilder().
		Engine(dkrSpec)

	return builder.BuildOrDie(), nil
}

func (d *DuckDBTranslator) dockerEngine(origin *models.SpecConfig) (*models.SpecConfig, error) {
	// It'd be nice to use pkg/executor/docker/types/EngineSpec here, but it
	// would mean adding a dependency on yet another package.
	cmd, cmdFound := origin.Params["Command"]
	args, argsFound := origin.Params["Arguments"]

	if !cmdFound || !argsFound {
		return nil, ErrMissingParameters("duckdb")
	}

	params := []string{}

	params = append(params, cmd.(string))
	params = append(params, args.([]string)...)

	spec := models.NewSpecConfig(models.EngineDocker)
	spec.Params["Image"] = "bacalhauproject/exec-duckdb:0.1"
	spec.Params["Entrypoint"] = []string{}
	spec.Params["Parameters"] = params
	spec.Params["EnvironmentVariables"] = []string{}
	spec.Params["WorkingDirectory"] = ""

	return spec, nil
}
