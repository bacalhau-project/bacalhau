package system

// event names - i.e. "this just happened"
const JOB_EVENT_CREATED = "job_created"
const JOB_EVENT_UPDATED = "job_updated"
const JOB_EVENT_RUN = "job_run"

// const JOB_EVENT_BID_CREATED = "bid_created"
// const JOB_EVENT_BID_ACCEPTED = "bid_accepted"
// const JOB_EVENT_BID_REJECTED = "bid_rejected"

// job states - these will be collected per host against a job
const JOB_STATE_BIDDING = "bidding"
const JOB_STATE_RUNNING = "running"
const JOB_STATE_ERROR = "error"
const JOB_STATE_COMPLETE = "complete"
