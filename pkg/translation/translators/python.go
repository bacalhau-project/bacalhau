package translators

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

var PythonPackageDomains = []string{
	"pypi.python.org",
	"pypi.org",
	"pythonhosted.org",
	"repo.anaconda.com",
	"repo.continuum.io",
	"conda.anaconda.org",
}

type PythonTranslator struct{}

func (p *PythonTranslator) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (p *PythonTranslator) Translate(original *models.Task) (*models.Task, error) {
	builder := original.
		ToBuilder().
		Engine(p.dockerEngine(original.Engine))

	original.Network = &models.NetworkConfig{
		Type:    models.NetworkHTTP,
		Domains: PythonPackageDomains,
	}

	return builder.BuildOrDie(), nil
}

func (p *PythonTranslator) dockerEngine(origin *models.SpecConfig) *models.SpecConfig {
	// It'd be nice to use pkg/executor/docker/types/EngineSpec here, but it
	// would mean adding a dependency on yet another package.
	cmd := origin.Params["Command"].(string)
	args := origin.Params["Arguments"].([]string)

	params := []string{
		"/build/launcher.py", "--",
	}

	params = append(params, cmd)
	params = append(params, args...)

	spec := models.NewSpecConfig(models.EngineDocker)
	spec.Params["Image"] = "bacalhauproject/exec-python-3.11:0.1"
	spec.Params["Entrypoint"] = []string{}
	spec.Params["Parameters"] = params
	spec.Params["EnvironmentVariables"] = []string{}
	spec.Params["WorkingDirectory"] = ""

	return spec
}
