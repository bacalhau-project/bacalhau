package handlerwrapper

type HTTPRequestInfo struct {
	JobID      string `json:"job_id,omitempty"` // bacalhau job id
	URI        string `json:"uri"`              // GET etc.
	Method     string `json:"method"`
	StatusCode int    `json:"status_code"` // response code, like 200, 404
	Size       int64  `json:"size"`        // number of bytes of the response sent
	Duration   int64  `json:"duration"`    // how long did it take to

	NodeID    string `json:"node_id"`             // bacalhau node id
	ClientID  string `json:"client_id,omitempty"` // bacalhau client id
	Referer   string `json:"referer,omitempty"`
	Ipaddr    string `json:"ipaddr"`
	UserAgent string `json:"user_agent"`
}

type RequestInfoHandler interface {
	Handle(*HTTPRequestInfo)
}
