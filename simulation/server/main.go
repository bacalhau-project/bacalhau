package main

import "fmt"

/*

Interfaces:

- can we use the in-memory noop stack?
- how do we add wallet balances to the noop stack? transport identity, transport is the wallet
	transportId -> $BACAL
- don't want libp2p, want a transport that is our simulation (of a future smart contract)
- our requestor node and compute node connected to a thing that's driving... wallet balance connected to it

- compute node for noop executor & factorial thing is fn, lie or not
- accept job, bid on job... is real code


swap out:
- use noop executor for custom fn as job
- let's use a Simulator transport which has a wallet balance associated & drive events thru REST API
- clients/agents are bits of code in any language that trigger things to happen...? submit job, bid on job, transport...

e.g
want to be nefarious requestor node, want compute node to react to me realistically
bacalhau network with compute nodes ready to react, with noop executor, and a requestor that is not normal...
simulator stack, rest API based transport
compute nodes attached, jobs arrive & bids accepted, they're ...

so the transport is a websocket?

make event happen, POST to rest endpoint, transport subscribe mechanism is listen to websocket


Transport: REST API + websocket + events + normal pure nodes stood up against, ... a lot of code already exists
noop stack + transport

to introduce nefarious writing clients to rest/ws

new transport: REST transport
could we actually just use libp2p? simulator libp2p transport w/wallet balances

nefarious agents that just join devstack ...
simulator libp2p transport that can do what we want

---

when we have a smart contract

imagine everything is in the smart contract, ...
smart contract has controller functions
DATASTORE interface is read/write to smart contract
TRANSPORT is listen to events from smart contract
datastore mutations trigger events

central thing: the thing that has wallet balances that we call controller methods on --> broadcast events
rest server that _IS_ the smart contract...
datastore is calls to rest API
and replacing transport that listens to websocket events out of the websocket API
simulating having smart contract in architecture

---

upside = introduce a centralized component, doing work that builds towards smart contract in the architecture
pair on building - mock ethereum with balances (rest API), put into architecture of bacalhau
most of what we need for a simulation and short of hardhat, done plumbing of smart contract in

how do we make a nefarious client appear? custom code that subscribes to events - simulation endpoint

* write smart contract mock (RPC call)
	- do that this afternoon

*/

func main() {

	fmt.Println("hello")
}
