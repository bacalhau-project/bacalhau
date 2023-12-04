package capacity

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/rs/zerolog/log"
)

// ToolBasedProvider will run an external tool and parse the results in a
// tool-specific way into a models.Resources instance.
//
// The tool is not required to return values for all fields. Any fields that
// aren't covered by the tool will have a zero value.
type ToolBasedProvider struct {
	Command  string
	Provides string
	Args     []string
	Parser   func(io.Reader) (models.Resources, error)
}

// GetAvailableCapacity implements Provider.
func (tool *ToolBasedProvider) GetAvailableCapacity(ctx context.Context) (models.Resources, error) {
	toolPath, err := exec.LookPath(tool.Command)
	if err != nil {
		// If the tool is not installed, we can't know the number of GPUs.
		// It is not an error to assume zero.
		log.Ctx(ctx).Info().Msgf("cannot inspect system %s: %s not installed. %s will not be used.", tool.Provides, tool.Command, tool.Provides)
		return models.Resources{}, nil
	}

	cmd := exec.Command(toolPath, tool.Args...)
	resp, err := cmd.Output()
	if err != nil {
		// we won't error here since some hosts may have the tool installed but
		// in a misconfigured state e.g. their drivers are missing, the smi
		// can't communicate with the drivers, etc. instead we provide a warning
		// show the args to the command we tried and its response.
		// motivation: https://expanso.atlassian.net/browse/GDAY-90
		log.Warn().Err(err).
			Str("command", fmt.Sprintf("%s %v", toolPath, tool.Args)).
			Str("response", string(resp)).
			Msgf("inspecting system failed")
		return models.Resources{}, nil
	}

	return tool.Parser(bytes.NewReader(resp))
}

var _ Provider = (*ToolBasedProvider)(nil)
