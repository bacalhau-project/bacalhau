package system

// event names - i.e. "this just happened"
const JOB_EVENT_CREATED = "job_created"
const JOB_EVENT_DEAL_UPDATED = "deal_updated"
const JOB_EVENT_BID = "bid"
const JOB_EVENT_BID_ACCEPTED = "bid_accepted"
const JOB_EVENT_BID_REJECTED = "bid_rejected"
const JOB_EVENT_RESULTS = "results"
const JOB_EVENT_RESULTS_ACCEPTED = "results_accepted"
const JOB_EVENT_RESULTS_REJECTED = "results_rejected"
const JOB_EVENT_ERROR = "error"

// job states - these will be collected per host against a job
const JOB_STATE_BIDDING = "bidding"
const JOB_STATE_BID_REJECTED = "bid_rejected"
const JOB_STATE_RUNNING = "running"
const JOB_STATE_ERROR = "error"
const JOB_STATE_COMPLETE = "complete"
const JOB_STATE_RESULTS_ACCEPTED = "accepted"
const JOB_STATE_RESULTS_REJECTED = "rejected"
