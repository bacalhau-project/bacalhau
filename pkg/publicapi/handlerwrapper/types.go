package handlerwrapper

import "context"

type HTTPRequestInfo struct {
	JobID      string `json:"jobID,omitempty"` // bacalhau job id
	URI        string `json:"URI"`             // GET etc.
	Method     string `json:"method"`
	StatusCode int    `json:"statusCode"` // response code, like 200, 404
	Size       int64  `json:"size"`       // number of bytes of the response sent
	Duration   int64  `json:"duration"`   // how long did it take to

	NodeID    string `json:"nodeID"`             // bacalhau node id
	ClientID  string `json:"clientID,omitempty"` // bacalhau client id
	Referer   string `json:"referer,omitempty"`
	Ipaddr    string `json:"IPaddr"`
	UserAgent string `json:"userAgent"`
}

type RequestInfoHandler interface {
	Handle(context.Context, *HTTPRequestInfo)
}
