package supervisor

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute/executor/environment"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type SupervisedTask struct {
	engine      string
	execution   *models.Execution
	environment *environment.Environment
}
