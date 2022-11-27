package simulator

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

// wallets get initialized with ₾10,000
// at < 1,000 we stop them putting any more messages on the network
// before they deposit more

const MIN_WALLET = 1000

type walletsModel struct {
	// keep track of which wallet address "owns" which job
	// the "ClientID" is only submitted for the create event
	// so we "remember" the ClientID for the job id
	jobOwners map[string]string

	// keep track of wallet balances - STUB (for Luke to fill in)
	balances map[string]uint64

	// keep track of a payment channel balance PER JOB, a.k.a per-job escrow
	escrow map[string]uint64

	// the local DB instance we can use to query state
	localDB localdb.LocalDB
}

func newWalletsModel(localDB localdb.LocalDB) *walletsModel {
	return &walletsModel{
		jobOwners: map[string]string{},
		balances:  map[string]uint64{},
		escrow:    map[string]uint64{},
		localDB:   localDB,
	}
}

// interpret each event as it comes in and adjust the wallet balances accordingly
// (as well as interrogate the localDB state)
func (wallets *walletsModel) addEvent(event model.JobEvent) error {
	log.Info().Msgf("SIM: wallets model received event %+v", event.EventName.String())

	/*
	   C: Created
	   S: Bid
	   C: Escrow & BidAccepted
	   S: ResultsProposed
	   C: ResultsAccepted & Payout (or not yet?)
	   C: ResultsPublished & Payout (now? can the smart contract verify the publishing?)
	*/

	switch event.EventName {
	case model.JobEventCreated:
		// C->S: Created
		return wallets.created(event)
	case model.JobEventBid:
		// S->C: Bid
		return wallets.bid(event)
	case model.JobEventBidAccepted:
		// C->S: Escrow & BidAccepted
		return wallets.bidAccepted(event)
	case model.JobEventResultsProposed:
		// S->C: ResultsProposed
		return wallets.resultsProposed(event)
	case model.JobEventResultsAccepted:
		// C->S: ResultsAccepted
		return wallets.resultsAccepted(event)
	case model.JobEventResultsPublished:
		// C->S: ResultsPublished
		return wallets.resultsPublished(event)
	}
	return nil
}

func (wallets *walletsModel) created(event model.JobEvent) error {
	log.Info().Msgf("SIM: received create event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	wallets.jobOwners[event.JobID] = event.ClientID
	// if we want to use the requester node as the wallet address then it's this
	//wallets.jobOwners[event.JobID] = event.SourceNodeID
	return nil
}

// an example of an event handler that maps the wallet address that "owns" the job
// and uses the state resolver to query the local DB for the current state of the job
func (wallets *walletsModel) bid(event model.JobEvent) error {
	ctx := context.Background()
	walletAddress := wallets.jobOwners[event.JobID]
	log.Info().Msgf("SIM: received bid event for job id: %s wallet address: %s\n", event.JobID, walletAddress)

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

func (wallets *walletsModel) bidAccepted(event model.JobEvent) error {
	log.Info().Msgf("SIM: received bidAccepted event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}

func (wallets *walletsModel) resultsProposed(event model.JobEvent) error {
	log.Info().Msgf("SIM: received resultsProposed event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}

func (wallets *walletsModel) resultsAccepted(event model.JobEvent) error {
	log.Info().Msgf("SIM: received resultsAccepted event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}

func (wallets *walletsModel) resultsPublished(event model.JobEvent) error {
	log.Info().Msgf("SIM: received resultsPublished event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}
