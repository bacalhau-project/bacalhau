package simulator

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type walletsModel struct {
	// keep track of which wallet address "owns" which job
	// the "ClientID" is only submitted for the create event
	// so we "remember" the ClientID for the job id
	jobOwners map[string]string

	// keep track of wallet balances - STUB (for Luke to fill in)
	balances map[string]uint64

	// the local DB instance we can use to query state
	localDB localdb.LocalDB
}

func newWalletsModel(localDB localdb.LocalDB) *walletsModel {
	return &walletsModel{
		jobOwners: map[string]string{},
		balances:  map[string]uint64{},
		localDB:   localDB,
	}
}

// interpret each event as it comes in and adjust the wallet balances accordingly
// (as well as interrogate the localDB state)
func (wallets *walletsModel) addEvent(event model.JobEvent) error {
	switch event.EventName {
	case model.JobEventCreated:
		return wallets.created(event)
	case model.JobEventBid:
		return wallets.bid(event)
	}
	return nil
}

func (wallets *walletsModel) created(event model.JobEvent) error {
	fmt.Printf("received create event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	wallets.jobOwners[event.JobID] = event.ClientID
	return nil
}

// an example of an event handler that maps the wallet address that "owns" the job
// and uses the state resolver to query the local DB for the current state of the job
func (wallets *walletsModel) bid(event model.JobEvent) error {
	ctx := context.Background()
	walletAddress := wallets.jobOwners[event.JobID]
	fmt.Printf("received bid event for job id: %s wallet address: %s\n", event.JobID, walletAddress)

	// here are examples of using the state resolver to query the localDB
	_, err := wallets.localDB.GetJob(ctx, event.JobID)
	if err != nil {
		return err
	}

	_, err = wallets.localDB.GetJobState(ctx, event.JobID)
	if err != nil {
		return err
	}

	return nil
}
