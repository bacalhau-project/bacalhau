package handlerwrapper

import "context"

type HTTPRequestInfo struct {
	JobID      string `json:"jobId,omitempty"` // bacalhau job id
	URI        string `json:"uri"`             // GET etc.
	Method     string `json:"method"`
	StatusCode int    `json:"statusCode"` // response code, like 200, 404
	Size       int64  `json:"size"`       // number of bytes of the response sent
	Duration   int64  `json:"duration"`   // how long did it take to

	NodeID    string `json:"nodeId"`             // bacalhau node id
	ClientID  string `json:"clientId,omitempty"` // bacalhau client id
	Referer   string `json:"referer,omitempty"`
	Ipaddr    string `json:"ipAddr"`
	UserAgent string `json:"userAgent"`
}

type RequestInfoHandler interface {
	Handle(context.Context, *HTTPRequestInfo)
}
