package system

// event names - i.e. "this just happened"
const JOB_EVENT_CREATED = "created"
const JOB_EVENT_UPDATED = "updated"
const JOB_EVENT_BID_ACCEPTED = "bid_accepted"

// job states - these will be collected per host against a job
const JOB_STATE_RUNNING = "running"
const JOB_STATE_ERROR = "error"
const JOB_STATE_COMPLETE = "complete"
