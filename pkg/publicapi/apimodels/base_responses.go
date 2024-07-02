package apimodels

type BaseResponse struct{}

// Normalize normalizes the response
func (o *BaseResponse) Normalize() {
	// noop
}

type BasePutResponse struct {
	BaseResponse `json:",omitempty,inline" yaml:",omitempty,inline"`
}

type BasePostResponse struct {
	BaseResponse `json:",omitempty,inline" yaml:",omitempty,inline"`
}

type BaseGetResponse struct {
	BaseResponse `json:",omitempty,inline" yaml:",omitempty,inline"`
}

type BaseListResponse struct {
	BaseGetResponse `json:",omitempty,inline" yaml:",omitempty,inline"`
	NextToken       string `json:"NextToken"`
}

func (o *BaseListResponse) GetNextToken() string { return o.NextToken }

// compile time check for interface implementation
var (
	_ Response     = (*BaseResponse)(nil)
	_ PutResponse  = (*BasePutResponse)(nil)
	_ PostResponse = (*BasePostResponse)(nil)
	_ GetResponse  = (*BaseGetResponse)(nil)
	_ ListResponse = (*BaseListResponse)(nil)
)
