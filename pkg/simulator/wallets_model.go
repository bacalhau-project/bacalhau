package simulator

import (
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
)

// wallets get initialized with ₾1,000
// at < 100 we stop them putting any more messages on the network
// before they deposit more
// we also slash by this amount - it ensures the nodes always have this much at
// stake

const MinWallet = 100 //nolint:gomnd

type walletsModel struct {
	// keep track of which wallet address "owns" which job
	// the "ClientID" is only submitted for the create event
	// so we "remember" the ClientID for the job id
	jobOwners map[string]string

	// keep track of wallet balances - STUB (for Luke to fill in)
	balances map[string]int64

	// keep track of a payment channel balance PER JOB, a.k.a per-job escrow
	// NB: in the real implementation, this would be indexed on (client, server,
	// job) and the payment channel would persist on the (client, server) prefix
	// of the key
	escrow map[string]int64

	// don't trust the server publishing the result to mean the client accepted it
	// mark in the smart contract that the client accepted it, instead
	accepted map[string]bool

	// money mutex - hold this when adding/subtracting to/from balances and
	// escrow channels
	moneyMutex sync.Mutex
}

func newWalletsModel() *walletsModel {
	w := &walletsModel{
		jobOwners: map[string]string{},
		balances:  map[string]int64{},
		escrow:    map[string]int64{},
		accepted:  map[string]bool{},
	}
	go w.logWallets()
	return w
}

func (wallets *walletsModel) logWallets() {
	for {
		log.Info().Msg("======== WALLET BALANCES =========")
		spew.Dump(wallets.balances)
		log.Info().Msg("======== ESCROW BALANCES =========")
		spew.Dump(wallets.escrow)
		time.Sleep(10 * time.Second) //nolint:gomnd
	}
}

// For now, annotate message types that we know come from requestor nodes as
// belonging to a requestor wallet, and similarly for compute nodes.
// This won't be necessary once
func fromClient(event model.JobEvent) model.JobEvent {
	event.SourceNodeID = event.SourceNodeID + "-requestor"
	if event.TargetNodeID != "" {
		event.TargetNodeID = event.TargetNodeID + "-computenode"
	}
	return event
}

func fromServer(event model.JobEvent) model.JobEvent {
	event.SourceNodeID = event.SourceNodeID + "-computenode"
	if event.TargetNodeID != "" {
		event.TargetNodeID = event.TargetNodeID + "-requestornode"
	}
	return event
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
	   C: ResultsAccepted
	   S: ResultsPublished & Payout
	*/

	// TODO: factor actual logic into fns below

	switch event.EventName {
	case model.JobEventCreated:
		// C->S: Created
		return wallets.created(fromClient(event))
	case model.JobEventBid:
		// S->C: Bid
		return wallets.bid(fromServer(event))
	case model.JobEventBidAccepted:
		// C->S: Escrow & BidAccepted
		event = fromClient(event)
		client := wallets.jobOwners[event.JobID]
		server := event.TargetNodeID
		// TODO: price in job spec! verify price per hour!
		// TODO: the client itself should escrow the funds, not the smart contract?
		err := wallets.escrowFunds(client, server, event.JobID, 33) //nolint:gomnd
		if err != nil {
			return err
		}
		// TODO: server should actually check that funds got escrowed?
		return wallets.bidAccepted(event)
	case model.JobEventResultsProposed:
		// S->C: ResultsProposed
		return wallets.resultsProposed(fromServer(event))
	case model.JobEventResultsAccepted:
		// C->S: ResultsAccepted
		event = fromClient(event)
		client := wallets.jobOwners[event.JobID]
		server := event.TargetNodeID
		escrowID := escrowID(client, server, event.JobID)
		wallets.accepted[escrowID] = true
		return wallets.resultsAccepted(event)
	case model.JobEventResultsRejected:
		event = fromClient(event)
		client := wallets.jobOwners[event.JobID]
		server := event.TargetNodeID
		err := wallets.refundAndSlash(client, server, event.JobID)
		if err != nil {
			return err
		}
	case model.JobEventResultsPublished:
		// S->C: ResultsPublished
		event = fromServer(event)
		client := wallets.jobOwners[event.JobID]
		server := event.SourceNodeID
		escrowID := escrowID(client, server, event.JobID)
		if !wallets.accepted[escrowID] {
			return fmt.Errorf(
				"tried to release escrow on job %s that was not accepted! "+
					"(on message from %s). naughty server",
				escrowID,
				server,
			)
		}
		err := wallets.releaseEscrow(client, server, event.JobID)
		if err != nil {
			return err
		}
		return wallets.resultsPublished(event)
	}
	return nil
}

func (wallets *walletsModel) created(event model.JobEvent) error {
	log.Info().Msgf("SIM: received create event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	// wallets.jobOwners[event.JobID] = event.ClientID
	// if we want to use the requester node as the wallet address then it's this
	wallets.jobOwners[event.JobID] = event.SourceNodeID
	return nil
}

func (wallets *walletsModel) checkWallet(event model.JobEvent) error {
	log.Info().Msgf("--> SIM(%s): ClientID: %s, SourceNodeID: %s", event.EventName.String(), event.ClientID, event.SourceNodeID)
	walletID := event.SourceNodeID
	wallets.ensureWallet(walletID)
	if wallets.balances[walletID] < MinWallet {
		return fmt.Errorf("wallet %s fell below min balance, sorry", walletID)
	}
	return nil
}

func (wallets *walletsModel) ensureWallet(wallet string) {
	if _, ok := wallets.balances[wallet]; !ok {
		wallets.balances[wallet] = 1000
	}
}

func escrowID(client, server, jobID string) string {
	return fmt.Sprintf("%s -> %s; jobID=%s", client, server, jobID)
}

func (wallets *walletsModel) escrowFunds(client, server, jobID string, amount int64) error {
	wallets.moneyMutex.Lock()
	defer wallets.moneyMutex.Unlock()
	wallet, ok := wallets.jobOwners[jobID]
	if !ok {
		return fmt.Errorf("job %s not found", jobID)
	}
	wallets.ensureWallet(wallet)
	if wallets.balances[wallet]-MinWallet < amount {
		return fmt.Errorf(
			"wallet %s has insufficient funds to escrow %d, "+
				"taking into account minimum wallet balance of %d",
			wallet,
			amount,
			MinWallet,
		)
	}
	wallets.balances[wallet] -= amount
	wallets.escrow[escrowID(client, server, jobID)] += amount
	return nil
}

func (wallets *walletsModel) releaseEscrow(client, server, jobID string) error {
	wallets.moneyMutex.Lock()
	defer wallets.moneyMutex.Unlock()
	escrowID := escrowID(client, server, jobID)
	amount, ok := wallets.escrow[escrowID]
	if !ok {
		return fmt.Errorf("no escrow found for %s", escrowID)
	}
	wallets.ensureWallet(server)
	wallets.balances[server] += amount
	delete(wallets.escrow, escrowID)
	return nil
}

func (wallets *walletsModel) refundAndSlash(client, server, jobID string) error {
	wallets.moneyMutex.Lock()
	defer wallets.moneyMutex.Unlock()
	escrowID := escrowID(client, server, jobID)
	amount, ok := wallets.escrow[escrowID]
	if !ok {
		return fmt.Errorf("no escrow found for %s", escrowID)
	}
	wallets.ensureWallet(client)
	wallets.ensureWallet(server)
	wallets.balances[client] += amount
	delete(wallets.escrow, escrowID)

	// and slash!
	log.Info().Msgf(`
 __________________________
< YOU GOT SLASHED %s >
 --------------------------
        \   ^__^
         \  (oo)\_______
            (__)\       )\/\
                ||----w |
                ||     ||
	`, server)
	// NB: the following is slash and burn
	wallets.balances[server] -= MinWallet
	return nil
}

// an example of an event handler that maps the wallet address that "owns" the job
// and uses the state resolver to query the local DB for the current state of the job
func (wallets *walletsModel) bid(event model.JobEvent) error {
	err := wallets.checkWallet(event)
	if err != nil {
		return err
	}

	walletAddress := wallets.jobOwners[event.JobID]
	wallets.ensureWallet(walletAddress)
	log.Info().Msgf("SIM: received bid event for job id: %s wallet address: %s\n", event.JobID, walletAddress)
	return nil
}

func (wallets *walletsModel) bidAccepted(event model.JobEvent) error {
	err := wallets.checkWallet(event)
	if err != nil {
		return err
	}

	log.Info().Msgf("SIM: received bidAccepted event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}

func (wallets *walletsModel) resultsProposed(event model.JobEvent) error {
	err := wallets.checkWallet(event)
	if err != nil {
		return err
	}

	log.Info().Msgf("SIM: received resultsProposed event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}

func (wallets *walletsModel) resultsAccepted(event model.JobEvent) error {
	err := wallets.checkWallet(event)
	if err != nil {
		return err
	}

	log.Info().Msgf("SIM: received resultsAccepted event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}

func (wallets *walletsModel) resultsPublished(event model.JobEvent) error {
	err := wallets.checkWallet(event)
	if err != nil {
		return err
	}

	log.Info().Msgf("SIM: received resultsPublished event for job id: %s wallet address: %s\n", event.JobID, event.ClientID)
	return nil
}
