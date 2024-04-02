package translators

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util"
)

const DuckDBImage = "bacalhauproject/exec-duckdb:0.2"

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
		Meta(models.MetaTranslatedBy, "translators/duckdb").
		Engine(dkrSpec)

	return builder.BuildOrDie(), nil
}

func (d *DuckDBTranslator) dockerEngine(origin *models.SpecConfig) (*models.SpecConfig, error) {
	// It'd be nice to use pkg/executor/docker/types/EngineSpec here, but it
	// would mean adding a dependency on yet another package.
	cmd := origin.Params["Command"].(string)
	args, err := util.InterfaceToStringArray(origin.Params["Arguments"])
	if err != nil {
		return nil, err
	}

	params := []string{}

	params = append(params, cmd)
	params = append(params, args...)

	spec := models.NewSpecConfig(models.EngineDocker)
	spec.Params["Image"] = DuckDBImage
	spec.Params["Entrypoint"] = []string{}
	spec.Params["Parameters"] = params
	spec.Params["EnvironmentVariables"] = []string{}
	spec.Params["WorkingDirectory"] = ""

	return spec, nil
}
