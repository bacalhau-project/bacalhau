package apimodels

import (
	"context"
	"strconv"
)

// BaseRequest is the base request used for all requests
type BaseRequest struct {
	Namespace string            `query:"namespace"`
	Headers   map[string]string `query:"-"`
	Context   context.Context   `query:"-"`

	// A good place to define other fields that are common to all requests,
	// such as auth tokens
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *BaseRequest) ToHTTPRequest() *HTTPRequest {
	r := NewHTTPRequest()

	if o.Namespace != "" {
		r.Params.Set("namespace", o.Namespace)
	}
	if o.Context != nil {
		r.Ctx = o.Context
	}

	for k, v := range o.Headers {
		r.Header.Set(k, v)
	}

	return r
}

// BasePutRequest is the base request used for all put requests
type BasePutRequest struct {
	BaseRequest
	IdempotencyToken string `query:"idempotency_token"`
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *BasePutRequest) ToHTTPRequest() *HTTPRequest {
	r := o.BaseRequest.ToHTTPRequest()

	if o.IdempotencyToken != "" {
		r.Params.Set("idempotency_token", o.IdempotencyToken)
	}
	return r
}

// BaseGetRequest is the base request used for all get requests
type BaseGetRequest struct {
	BaseRequest
}

// BaseListRequest is the base request used for all list requests
type BaseListRequest struct {
	BaseGetRequest
	Limit     uint32 `query:"limit"`
	NextToken string `query:"next_token"`
	OrderBy   string `query:"order_by"`
	Reverse   bool   `query:"reverse"`
}

// ToHTTPRequest is used to convert the request to an HTTP request
func (o *BaseListRequest) ToHTTPRequest() *HTTPRequest {
	r := o.BaseGetRequest.ToHTTPRequest()

	if o.Limit != 0 {
		r.Params.Set("limit", strconv.Itoa(int(o.Limit)))
	}
	if o.NextToken != "" {
		r.Params.Set("next_token", o.NextToken)
	}
	if o.OrderBy != "" {
		r.Params.Set("order_by", o.OrderBy)
	}
	if o.Reverse {
		r.Params.Set("reverse", "true")
	}
	return r
}

// compile time check for interface implementation
var _ Request = (*BaseRequest)(nil)
var _ PutRequest = (*BasePutRequest)(nil)
var _ GetRequest = (*BaseGetRequest)(nil)
var _ ListRequest = (*BaseListRequest)(nil)
