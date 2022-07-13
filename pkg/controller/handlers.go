package controller

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
)

// handlers are the interactions with the data store that happen both on the initial call of
// a public controller method and also as a result of event arriving across the wire
// in both cases the data is updated in the datastore

// called by both self.SubmitJob and eventReader.JobEventCreated
func (ctrl *Controller) handleJobCreated(ctx context.Context, ev executor.JobEvent) error {
	job := constructJob(ev)
	err := ctrl.datastore.AddJob(ctx, job)
	if err != nil {
		return err
	}
	err = ctrl.datastore.AddEvent(ctx, ev.JobID, ev)
	if err != nil {
		return err
	}
	return nil
}

// called by both self.UpdateDeal and eventReader.JobEventDealUpdated
func (ctrl *Controller) handleDealUpdated(ctx context.Context, ev executor.JobEvent) error {
	err := ctrl.datastore.UpdateJobDeal(ctx, ev.JobID, ev.JobDeal)
	if err != nil {
		return err
	}

	err = ctrl.datastore.AddEvent(ctx, ev.JobID, ev)
	if err != nil {
		return err
	}

	return nil
}
