# architecture

 * controller
   * state machine
   * data store
   * transport
 
## controller

The controller is how you get jobs and change the state of jobs.

It keeps a consistent view of local state of jobs (via the data store) and broadcasts the state to other nodes (via the transport).

## controller -> state machine

To initiate the change in the state of a job - the corresponding controller method will be called (e.g. BidJob, AcceptBid)

The state machine will check is the current transition for a job valid and:

 * update the local data store
 * broadcast the new state to other nodes

## controller -> data store

The data store will:

 * persist network job state across restarts
 * give a consistent local view of network and local job state
 * allow control loops to update the local state of jobs

**network state** is the state of jobs as far as the network is concerned.

**local state** is the state of jobs as far as only the local node is concerned.

An example of network state is "node 123 has bid on job 456".

An example of local state is "I have selected job 456 but am not ready to bid on it yet".

## controller -> transport

The job of the transport is to broadcast the state of jobs to other nodes.

When events are recieved from other nodes on the network - the transport will update the data store before emitting a controller event.