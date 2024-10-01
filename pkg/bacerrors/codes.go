package bacerrors

type ErrorCode string

const (
	BadRequestError    ErrorCode = "BadRequest"
	InternalError      ErrorCode = "InternalError"
	NotFoundError      ErrorCode = "NotFound"
	TimeOutError       ErrorCode = "TimeOut"
	UnauthorizedError  ErrorCode = "Unauthorized"
	ServiceUnavailable ErrorCode = "ServiceUnavailable"
	NotImplemented     ErrorCode = "NotImplemented"
	ResourceExhausted  ErrorCode = "ResourceExhausted"
	ResourceInUse      ErrorCode = "ResourceInUse"
	VersionMismatch    ErrorCode = "VersionMismatch"
	ValidationError    ErrorCode = "ValidationError"
	TooManyRequests    ErrorCode = "TooManyRequests"
	NetworkFailure     ErrorCode = "NetworkFailure"
	ConfigurationError ErrorCode = "ConfigurationError"
	DatastoreFailure   ErrorCode = "DatastoreFailure"
	RequestCancelled   ErrorCode = "RequestCancelled"
	UnknownError       ErrorCode = "UnknownError"
)

func Code(code string) ErrorCode {
	return ErrorCode(code)
}
