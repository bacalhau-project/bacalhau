package handlerwrapper

import "context"

type HTTPRequestInfo struct {
	JobID      string `json:"JobID,omitempty"` // bacalhau job id
	URI        string `json:"URI"`             // GET etc.
	Method     string `json:"Method"`
	StatusCode int    `json:"StatusCode"` // response code, like 200, 404
	Size       int64  `json:"Size"`       // number of bytes of the response sent
	Duration   int64  `json:"Duration"`   // how long did it take to

	NodeID    string `json:"NodeID"`             // bacalhau node id
	ClientID  string `json:"ClientID,omitempty"` // bacalhau client id
	Referer   string `json:"Referer,omitempty"`
	Ipaddr    string `json:"Ipaddr"`
	UserAgent string `json:"UserAgent"`
}

type RequestInfoHandler interface {
	Handle(context.Context, *HTTPRequestInfo)
}
