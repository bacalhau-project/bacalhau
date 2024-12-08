package ncl

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

var (
	// ErrHandlerExists is returned when attempting to register a handler for a message type that already has one
	ErrHandlerExists = errors.New("handler already exists for message type")

	// ErrNoHandler is returned when no handler is found for a message type
	ErrNoHandler = errors.New("no handler found for message type")
)

type responder struct {
	nc      *nats.Conn
	config  ResponderConfig
	encoder *encoder

	handlers     map[string]RequestHandler
	subscription *nats.Subscription
	mu           sync.Mutex
}

// NewResponder creates a new responder instance
func NewResponder(nc *nats.Conn, config ResponderConfig) (Responder, error) {
	config.setDefaults()

	enc, err := newEncoder(encoderConfig{
		source:            config.Name,
		messageSerializer: config.MessageSerializer,
		messageRegistry:   config.MessageRegistry,
	})
	if err != nil {
		return nil, err
	}

	r := &responder{
		nc:       nc,
		config:   config,
		handlers: make(map[string]RequestHandler),
		encoder:  enc,
	}

	// Validate the subscriber
	if err := r.validate(); err != nil {
		return nil, fmt.Errorf("error validating responder: %w", err)
	}

	return r, nil
}

func (r *responder) validate() error {
	return errors.Join(
		validate.NotNil(r.nc, "NATS connection cannot be nil"),
		r.config.Validate(),
	)
}

// Listen registers a handler for a specific message type. When messages of this type
// are received, they will be dispatched to the provided handler. If this is the first
// handler being registered, it will create the NATS subscription.
// Returns ErrHandlerExists if a handler is already registered for this message type.
func (r *responder) Listen(ctx context.Context, messageType string, handler RequestHandler) error {
	if err := validate.NotBlank(messageType, "message type cannot be blank"); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for existing handler
	if _, exists := r.handlers[messageType]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerExists, messageType)
	}

	// Register the handler for this message type
	r.handlers[messageType] = handler
	log.Debug().Str("messageType", messageType).Msg("Registered new message handler")

	// Subscribe if this is our first handler
	if r.subscription == nil {
		sub, err := r.nc.Subscribe(r.config.Subject, r.handleRequest)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", r.config.Subject, err)
		}
		r.subscription = sub
		log.Debug().Str("subject", r.config.Subject).Msg("Created NATS subscription")
	}

	return nil
}

func (r *responder) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.subscription != nil && r.subscription.IsValid() {
		if err := r.subscription.Unsubscribe(); err != nil {
			return fmt.Errorf("error closing subscription: %w", err)
		}
		r.subscription = nil
	}

	// Clear handlers
	r.handlers = make(map[string]RequestHandler)
	return nil
}

// handleRequest is the NATS message callback that processes incoming requests.
// It:
// 1. Validates the message has a reply subject
// 2. Deserializes the request envelope
// 3. Finds the appropriate handler for the message type
// 4. Processes the request and sends back a response
// Any errors during processing result in an error response being sent back.
func (r *responder) handleRequest(msg *nats.Msg) {
	// Check for reply subject
	if msg.Reply == "" {
		log.Warn().
			Str("subject", msg.Subject).
			Msg("Received message without reply subject")
		return
	}

	// Create processing context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.config.ProcessingTimeout)
	defer cancel()

	// Deserialize request envelope
	request, err := r.encoder.decode(msg.Data)
	if err != nil {
		errorResponse := NewErrorResponse(StatusBadRequest, err.Error())
		r.sendErrorResponse(msg, errorResponse)
		return
	}

	// Get message type and find handler
	messageType := request.Metadata.Get(envelope.KeyMessageType)
	r.mu.Lock()
	handler, exists := r.handlers[messageType]
	r.mu.Unlock()

	if !exists {
		log.Warn().
			Str("messageType", messageType).
			Str("subject", msg.Subject).
			Msg("No handler registered for message type")
		errorResponse := NewErrorResponse(
			StatusNotFound, fmt.Errorf("no handler found for message type: %s", messageType).Error())
		r.sendErrorResponse(msg, errorResponse)
		return
	}

	// Process request with the appropriate handler
	response, err := handler.HandleRequest(ctx, request)
	if err != nil {
		errorResponse := NewErrorResponse(
			StatusServerError, fmt.Errorf("failed to process request: %w", err).Error())
		r.sendErrorResponse(msg, errorResponse)
		return
	}

	r.sendResponse(msg, response)
}

// sendResponse sends a response message back through NATS.
// It preserves correlation IDs and handles serialization of the response envelope.
func (r *responder) sendResponse(natsMsg *nats.Msg, response *envelope.Message) {
	// Preserve request correlation ID if present
	if reqID := natsMsg.Header.Get(KeyMessageID); reqID != "" {
		response.WithMetadataValue(KeyMessageID, reqID)
	}

	// Serialize response
	data, err := r.encoder.encode(response)
	if err != nil {
		errorResponse := NewErrorResponse(StatusServerError, err.Error())
		r.sendOrLogError(natsMsg, response, errorResponse)
		return
	}

	// Send response
	if err = natsMsg.RespondMsg(&nats.Msg{
		Data:   data,
		Header: response.Metadata.ToHeaders(),
	}); err != nil {
		log.Error().Err(err).
			Str("subject", natsMsg.Subject).
			Msg("Failed to send response")
	}
}

// sendErrorResponse is a convenience method to send an error response.
// It converts the ErrorResponse to an envelope before sending.
func (r *responder) sendErrorResponse(msg *nats.Msg, response ErrorResponse) {
	r.sendResponse(msg, response.ToEnvelope())
}

// sendOrLogError handles errors that occur while sending error responses.
// If we fail to send an error response, we log it instead of trying again
// to avoid potential infinite loops.
func (r *responder) sendOrLogError(msg *nats.Msg, originalResponse *envelope.Message, errorResponse ErrorResponse) {
	if originalResponse.Metadata.Get(envelope.KeyMessageType) == ErrorMessageType {
		log.Error().Msgf("failed to send error response to %s: %s", msg.Subject, originalResponse.Payload)
	} else {
		r.sendResponse(msg, originalResponse)
	}
}

// compile-time check for interface conformance
var _ Responder = (*responder)(nil)
