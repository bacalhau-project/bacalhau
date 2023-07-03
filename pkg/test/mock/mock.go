package mock

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/google/uuid"
)

func Eval() *model.Evaluation {
	now := time.Now().UTC().UnixNano()
	eval := &model.Evaluation{
		ID:         uuid.NewString(),
		Namespace:  model.DefaultNamespace,
		Priority:   50,
		Type:       model.JobTypeBatch,
		JobID:      uuid.NewString(),
		Status:     model.EvalStatusPending,
		CreateTime: now,
		ModifyTime: now,
	}
	return eval
}
