package stream

// Close error codes that follow these semantics:
// - 1000 Series: Normal and controlled shutdown scenarios. These are standard and expected reasons for closing a stream.
// - 4000 Series: Client-related errors or issues with the transmitted data, indicating problems that originate from the caller's side.
// - 5000 Series: Server-side errors, indicating problems that are internal to the server or the processing system.
const (
	// CloseNormalClosure Indicates that the stream was closed after completing its
	// intended purpose. No errors occurred, and the closure was planned.
	CloseNormalClosure = 1000
	// CloseGoingAway Used when a service is shutting down or a client
	// is disconnecting normally but unexpectedly
	CloseGoingAway = 1001
	// CloseUnsupportedData Indicates that data sent over an established stream is invalid,
	// corrupt, or cannot be processed. This is used after a stream has been successfully
	// established but encounters data-related issues.
	CloseUnsupportedData = 1003
	// CloseAbnormalClosure Indicates that a stream was closed unexpectedly,
	// without a known reason. This might be used when a connection is dropped
	// due to network issues, or a service crashes.
	CloseAbnormalClosure = 1006
	// CloseBadRequest Signifies that the initial request to establish a stream contained
	// invalid parameters or was malformed, preventing the stream from being established.
	CloseBadRequest = 4000
	// CloseInternalServerErr Used when an unexpected condition was encountered
	// by the server, preventing it from fulfilling the request. This signals issues
	// that are internal to the server or the processing system.
	CloseInternalServerErr = 5000
)
