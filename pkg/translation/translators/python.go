package translators

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/util"
	"golang.org/x/exp/maps"
)

// PythonPackageDomains lists all of the domains that might be needed to install
// dependencies at runtime.
var PythonPackageDomains = []string{
	"pypi.python.org",
	"pypi.org",
	"pythonhosted.org",
	"files.pythonhosted.org",
	"repo.anaconda.com",
	"repo.continuum.io",
	"conda.anaconda.org",
}

// SupportedPythonVersions maps the python version to the docker image that
// provides support for that version.
var SupportedPythonVersions = map[string]string{
	"3.11": "bacalhauproject/exec-python-3.11:0.5",
}

type PythonTranslator struct{}

func (p *PythonTranslator) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (p *PythonTranslator) Translate(original *models.Task) (*models.Task, error) {
	dkrSpec, err := p.dockerEngine(original.Engine)
	if err != nil {
		return nil, err
	}

	builder := original.
		ToBuilder().
		Meta(models.MetaTranslatedBy, "translators/python").
		Engine(dkrSpec)

	original.Network = &models.NetworkConfig{
		Type:    models.NetworkHTTP,
		Domains: PythonPackageDomains,
	}

	return builder.BuildOrDie(), nil
}

func (p *PythonTranslator) dockerEngine(origin *models.SpecConfig) (*models.SpecConfig, error) {
	// It'd be nice to use pkg/executor/docker/types/EngineSpec here, but it
	// would mean adding a dependency on yet another package.
	cmd := origin.Params["Command"].(string)
	args, err := util.InterfaceToStringArray(origin.Params["Arguments"])
	if err != nil {
		return nil, err
	}

	versionString := "3.11" // Default version
	version := origin.Params["Version"]
	if version != nil {
		versionString = version.(string)
	}

	image, err := getImageName(versionString)
	if err != nil {
		return nil, err
	}

	params := []string{
		"/build/launcher.py", "--",
	}

	params = append(params, cmd)
	params = append(params, args...)

	spec := models.NewSpecConfig(models.EngineDocker)
	spec.Params["Image"] = image
	spec.Params["Entrypoint"] = []string{}
	spec.Params["Parameters"] = params
	spec.Params["EnvironmentVariables"] = []string{}
	spec.Params["WorkingDirectory"] = ""

	return spec, nil
}

func getImageName(version string) (string, error) {
	image, found := SupportedPythonVersions[version]
	if !found {
		supported := ""
		versions := maps.Keys(SupportedPythonVersions)
		for i := range versions {
			supported += fmt.Sprintf("  * %s\n", versions[i])
		}
		return "", fmt.Errorf("unsupported python version: %s\nsupported versions are:\n%s", version, supported)
	}
	return image, nil
}
