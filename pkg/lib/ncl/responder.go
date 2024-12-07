package ncl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
	nc     *nats.Conn
	config ResponderConfig

	handlers     map[string]RequestHandler
	subscription *nats.Subscription
	mu           sync.Mutex
}

// NewResponder creates a new responder instance
func NewResponder(nc *nats.Conn, config ResponderConfig) (Responder, error) {
	config.setDefaults()

	r := &responder{
		nc:       nc,
		config:   config,
		handlers: make(map[string]RequestHandler),
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

// Listen registers a handler for a specific message type
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
	rMsg, err := r.config.MessageSerializer.Deserialize(msg.Data)
	if err != nil {
		errorResponse := NewErrorResponse(
			StatusBadRequest, fmt.Errorf("failed to deserialize request envelope: %w", err).Error())
		r.sendErrorResponse(msg, errorResponse)
		return
	}

	// Get message type and find handler
	messageType := rMsg.Metadata.Get(envelope.KeyMessageType)
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

	// Deserialize request payload
	request, err := r.config.MessageRegistry.Deserialize(rMsg)
	if err != nil {
		errorResponse := NewErrorResponse(
			StatusBadRequest, fmt.Errorf("failed to deserialize request payload: %w", err).Error())
		r.sendErrorResponse(msg, errorResponse)
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

func (r *responder) sendResponse(natsMsg *nats.Msg, response *envelope.Message) {
	// Add response metadata
	response.WithMetadataValue(KeySource, r.config.Name)
	response.WithMetadataValue(KeyEventTime, time.Now().Format(time.RFC3339))

	// Preserve request correlation ID if present
	if reqID := natsMsg.Header.Get(KeyMessageID); reqID != "" {
		response.WithMetadataValue(KeyMessageID, reqID)
	}

	// Serialize response
	rMsg, err := r.config.MessageRegistry.Serialize(response)
	if err != nil {
		errorResponse := NewErrorResponse(
			StatusServerError, fmt.Errorf("failed to serialize response payload: %w", err).Error())
		r.sendOrLogError(natsMsg, response, errorResponse)
		return
	}

	data, err := r.config.MessageSerializer.Serialize(rMsg)
	if err != nil {
		errorResponse := NewErrorResponse(
			StatusServerError, fmt.Errorf("failed to serialize response envelope: %w", err).Error())
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

func (r *responder) sendErrorResponse(msg *nats.Msg, response ErrorResponse) {
	r.sendResponse(msg, response.ToEnvelope())
}

// sendOrLogError
func (r *responder) sendOrLogError(msg *nats.Msg, originalResponse *envelope.Message, errorResponse ErrorResponse) {
	if originalResponse.Metadata.Get(envelope.KeyMessageType) == ErrorMessageType {
		log.Error().Msgf("failed to send error response to %s: %s", msg.Subject, originalResponse.Payload)
	} else {
		r.sendResponse(msg, originalResponse)
	}
}

// compile-time check for interface conformance
var _ Responder = (*responder)(nil)
