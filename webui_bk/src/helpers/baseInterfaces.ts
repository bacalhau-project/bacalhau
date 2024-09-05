// The interfaces in this file match those in base_requests.go

export interface Request {
    namespace?: string
}

export interface GetRequest extends Request {}

export interface ListRequest extends GetRequest {
    limit?: number
    next_token?: string
    order_by?: string
    reverse?: boolean
}

export interface ListResponse {
    NextToken: string
}
