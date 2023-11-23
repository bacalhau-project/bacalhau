package middleware

import (
	"context"
	"fmt"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const (
	// DropAcceptEncodingMiddlewareID uniquely identifies the middleware that drops the Accept-Encoding header.
	DropAcceptEncodingMiddlewareID = "DropAcceptEncodingHeader"

	// RestoreAcceptEncodingMiddlewareID uniquely identifies the middleware that restores the Accept-Encoding header.
	RestoreAcceptEncodingMiddlewareID = "RestoreAcceptEncodingHeader"

	// AcceptEncodingHeader is the HTTP header key for Accept-Encoding.
	AcceptEncodingHeader = "Accept-Encoding"

	// SigningMiddlewareID identifies the AWS request signing middleware.
	SigningMiddlewareID = "Signing"
)

// DropAcceptEncoding configures the S3 client options to include middleware
// that modifies the request headers to be compatible with S3-like services that
// do not support the Accept-Encoding header.
// It does this by dropping the Accept-Encoding header before the request is signed, and restoring it after the request is signed.
// https://stackoverflow.com/a/74382598
func DropAcceptEncoding(o *s3.Options) {
	o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
		// Return early if the signing middleware is not present, such as with PresignGetObject requests.
		_, ok := stack.Finalize.Get(SigningMiddlewareID)
		if !ok {
			return nil
		}

		// Insert the middleware to drop the Accept-Encoding header before the request is signed.
		if err := stack.Finalize.Insert(dropAcceptEncodingHeader, SigningMiddlewareID, middleware.Before); err != nil {
			return err
		}

		// Insert the middleware to restore the Accept-Encoding header after the request is signed.
		if err := stack.Finalize.Insert(restoreAcceptEncodingHeader, SigningMiddlewareID, middleware.After); err != nil {
			return err
		}

		return nil
	})
}

// dropAcceptEncodingHeader is a middleware function that removes the Accept-Encoding header from the request.
// This is necessary for compatibility with certain S3-like storage providers.
//
//nolint:lll
var dropAcceptEncodingHeader = middleware.FinalizeMiddlewareFunc(DropAcceptEncodingMiddlewareID,
	func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (
		out middleware.FinalizeOutput, metadata middleware.Metadata, err error,
	) {
		req, ok := in.Request.(*smithyhttp.Request)
		if !ok {
			// Return an error if the request type is unexpected.
			return out, metadata, &v4.SigningError{Err: fmt.Errorf("unexpected request middleware type %T", in.Request)}
		}

		// Store the current Accept-Encoding header value in the context.
		ae := req.Header.Get(AcceptEncodingHeader)
		ctx = setAcceptEncodingKey(ctx, ae)

		// Remove the Accept-Encoding header from the request.
		req.Header.Del(AcceptEncodingHeader)
		in.Request = req

		// Proceed with the next middleware in the stack.
		return next.HandleFinalize(ctx, in)
	},
)

// restoreAcceptEncodingHeader is a middleware function that restores the Accept-Encoding header to the request.
// This is done after the request is signed to maintain compatibility with certain S3-like storage providers.
//
//nolint:lll
var restoreAcceptEncodingHeader = middleware.FinalizeMiddlewareFunc(RestoreAcceptEncodingMiddlewareID,
	func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (
		out middleware.FinalizeOutput, metadata middleware.Metadata, err error,
	) {
		req, ok := in.Request.(*smithyhttp.Request)
		if !ok {
			// Return an error if the request type is unexpected.
			return out, metadata, &v4.SigningError{Err: fmt.Errorf("unexpected request middleware type %T", in.Request)}
		}

		// Retrieve the original Accept-Encoding header value from the context.
		ae := getAcceptEncodingKey(ctx)

		// Restore the Accept-Encoding header in the request.
		req.Header.Set(AcceptEncodingHeader, ae)
		in.Request = req

		// Proceed with the next middleware in the stack.
		return next.HandleFinalize(ctx, in)
	},
)

// acceptEncodingKey is a context key used for storing the Accept-Encoding header value.
type acceptEncodingKey struct{}

// getAcceptEncodingKey retrieves the Accept-Encoding header value from the context.
func getAcceptEncodingKey(ctx context.Context) (v string) {
	v, _ = middleware.GetStackValue(ctx, acceptEncodingKey{}).(string)
	return v
}

// setAcceptEncodingKey stores the Accept-Encoding header value in the context.
func setAcceptEncodingKey(ctx context.Context, value string) context.Context {
	return middleware.WithStackValue(ctx, acceptEncodingKey{}, value)
}
