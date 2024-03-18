package apimodels

type Request interface {
	// SetCredential is used to set the authorization token for the request
	SetCredential(*HTTPCredential)
	// ToHTTPRequest is used to convert the request to an HTTP request
	ToHTTPRequest() *HTTPRequest
}

type PutRequest interface {
	Request
}

type PostRequest interface {
	Request
}

type GetRequest interface {
	Request
}

type ListRequest interface {
	GetRequest
}

type Response interface {
	// Normalize normalizes the response
	Normalize()
}

type PutResponse interface {
	Response
}

type PostResponse interface {
	Response
}

type GetResponse interface {
	Response
}

type ListResponse interface {
	GetResponse

	// GetNextToken is the token used to indicate where to start paging
	// for queries that support paginated lists. To resume paging from
	// this point, pass this token in the next request
	GetNextToken() string
}
