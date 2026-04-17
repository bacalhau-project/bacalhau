package capacity

import (
	"bytes"
	"context"
	"io"
	"os/exec"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/pkg/errors"
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
		// If the tool is not installed, we can't know the available resources.
		return models.Resources{}, errors.Wrapf(err, "tool %q is not installed or not on PATH", tool.Command)
	}

	//nolint:gosec // G204: toolPath validated by exec.LookPath, Args from trusted config
	cmd := exec.CommandContext(ctx, toolPath, tool.Args...)
	resp, err := cmd.Output()
	if err != nil {
		return models.Resources{}, errors.Wrapf(err, "tool `%s %v` had bad exit and returned: %q", toolPath, tool.Args, string(resp))
	}

	resources, err := tool.Parser(bytes.NewReader(resp))
	if err != nil {
		return models.Resources{}, errors.Wrapf(err, "tool `%s %v` had unparsable output: %q", toolPath, tool.Args, string(resp))
	}

	return resources, nil
}

// ResourceTypes implements Subprovider.
func (tool *ToolBasedProvider) ResourceTypes() []string {
	return []string{tool.Provides}
}

var _ Provider = (*ToolBasedProvider)(nil)
