package requester

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/jobstore"
	"github.com/rs/zerolog/log"
)

type HousekeepingParams struct {
	Endpoint Endpoint
	JobStore jobstore.Store
	NodeID   string
	Interval time.Duration
}

type Housekeeping struct {
	endpoint Endpoint
	jobStore jobstore.Store
	nodeID   string
	interval time.Duration

	stopChannel chan struct{}
	stopOnce    sync.Once
}

func NewHousekeeping(params HousekeepingParams) *Housekeeping {
	h := &Housekeeping{
		endpoint:    params.Endpoint,
		jobStore:    params.JobStore,
		nodeID:      params.NodeID,
		interval:    params.Interval,
		stopChannel: make(chan struct{}),
	}

	go h.housekeepingBackgroundTask()
	return h
}

func (h *Housekeeping) housekeepingBackgroundTask() {
	ctx := context.Background()
	ticker := time.NewTicker(h.interval)
	for {
		select {
		case <-ticker.C:
			jobs, err := h.jobStore.GetInProgressJobs(ctx)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to get in progress jobs")
				continue
			}
			now := time.Now()
			for _, jobDescription := range jobs {
				// in case the job store is shared between multiple nodes, we only want to clean up jobs that are owned by this node
				if jobDescription.Job.Metadata.Requester.RequesterNodeID != h.nodeID {
					continue
				}
				// cancel jobs that have been in progress beyond the timeout period
				if now.Sub(jobDescription.State.CreateTime).Seconds() > jobDescription.Job.Spec.Timeout {
					log.Ctx(ctx).Info().Msgf("job %s timed out. Canceling", jobDescription.Job.Metadata.ID)
					go func(jobID string) {
						_, innerErr := h.endpoint.CancelJob(ctx, CancelJobRequest{
							JobID:  jobID,
							Reason: "timed out",
						})
						if innerErr != nil {
							log.Ctx(ctx).Err(innerErr).Msgf("failed to cancel job %s", jobID)
						}
					}(jobDescription.Job.Metadata.ID)
				}
			}
		case <-h.stopChannel:
			log.Ctx(ctx).Debug().Msg("stopped housekeeping task")
			ticker.Stop()
			return
		}
	}
}

func (h *Housekeeping) Stop() {
	h.stopOnce.Do(func() {
		h.stopChannel <- struct{}{}
	})
}
