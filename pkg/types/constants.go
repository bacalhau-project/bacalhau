package types

type JobEventType string

// event names - i.e. "this just happened"
const (
	JOB_EVENT_CREATED          JobEventType = "job_created"
	JOB_EVENT_DEAL_UPDATED     JobEventType = "deal_updated"
	JOB_EVENT_BID              JobEventType = "bid"
	JOB_EVENT_BID_ACCEPTED     JobEventType = "bid_accepted"
	JOB_EVENT_BID_REJECTED     JobEventType = "bid_rejected"
	JOB_EVENT_RESULTS          JobEventType = "results"
	JOB_EVENT_RESULTS_ACCEPTED JobEventType = "results_accepted"
	JOB_EVENT_RESULTS_REJECTED JobEventType = "results_rejected"
	JOB_EVENT_ERROR            JobEventType = "error"
)

type JobStateType string

// job states - these will be collected per host against a job
const (
	JOB_STATE_BIDDING      JobStateType = "bidding"
	JOB_STATE_BID_REJECTED JobStateType = "bid_rejected"
	JOB_STATE_RUNNING      JobStateType = "running"
	JOB_STATE_ERROR        JobStateType = "error"
	JOB_STATE_COMPLETE     JobStateType = "complete"
)
