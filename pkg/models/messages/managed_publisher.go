package messages

type ManagedPublisherPreSignURLRequest struct {
	BaseRequest
	ExecutionID string
	JobID       string
}

type ManagedPublisherPreSignURLResponse struct {
	BaseResponse
	ExecutionID  string
	JobID        string
	PreSignedURL string
}
