package stream

import (
	"context"
	"strconv"
	"time"
)

// StreamingMsgType represents the type of a streaming message.
type StreamingMsgType int

const (
	streamingMsgTypeUnknown StreamingMsgType = iota //nolint:unused
	// streamingMsgTypeData represents a data message.
	streamingMsgTypeData
	// streamingMsgTypeClose represents a close message.
	streamingMsgTypeClose
)

// StreamingMsg represents a streaming message that can be sent over NATS.
// It can be a data message or a close message.
type StreamingMsg struct {
	// Type is the type of the message.
	Type StreamingMsgType `json:"type"`
	// Data is the optional data payload. It is only used if Type is streamingMsgTypeData.
	Data []byte `json:"data,omitempty"`
	// CloseError is the optional close message. It is only used if Type is streamingMsgTypeClose.
	CloseError *CloseError `json:"closeError,omitempty"`
}

type Request struct {
	// ConsumerID is the connection id of the consumer streaming client originating the request.
	ConsumerID string `json:"consumerId"`
	// StreamId is the id of the stream being created.
	StreamID string `json:"streamId"`
	// HeartBeatSub is the heart beat subject where the producer client will send its heart beat.
	HeartBeatRequestSub string `json:"heartBeatRequestSub"`
	// Data represents request of different stream type. For example currently we support Log request
	// in that case it would be ExecutionLogRequest
	Data []byte `json:"body"`
}

// StreamInfo represents information about the stream.
type StreamInfo struct {
	// ID is the identifier of the stream.
	ID string
	// RequestSub is the subject on which the request for this stream was sent.
	RequestSub string
	// CreatedAt represents the time the stream was created.
	CreatedAt time.Time
	// Function to cancel the stream. This is useful in the event the consumer client
	// is no longer interested in the stream. The cancel function is inovked informing the
	// producer to no longer serve the stream.
	Cancel context.CancelFunc
}

// StreamProducerClientConfig represents the configuration of NATS based streaming
// client acting as a producer.
type StreamProducerClientConfig struct {
	// HeartBeatIntervalDuration represents the duration between two heart beats from the producer client
	// to consumer client.
	HeartBeatIntervalDuration time.Duration
	// HeartBeatRequestTimeout represents the time within which the producer client should receive the
	// response from the consumer client.
	HeartBeatRequestTimeout time.Duration
	// StreamCancellationBufferDuration represents the time interval for which consumer or producer client
	// should wait before killing the stream in case of race conditions on heart beats and request origination.
	StreamCancellationBufferDuration time.Duration
}

// StreamConsumerClientConfig represents the configuration of NATS based streaming
// client acting as a consumer.
type StreamConsumerClientConfig struct {
	StreamCancellationBufferDuration time.Duration
}

// HeartBeatRequest sent by producer client to the consumer client.
type HeartBeatRequest struct {
	// ActiveStreamIds is a map of active stream ids on producer client, where key is the RequestSubject, where
	// the original request to initiate a streaming connection was sent.
	ActiveStreamIds map[string][]string
}

// ConsumerHeartBeatResponse represents a heart beat response from the consumer client.
type ConsumerHeartBeatResponse struct {
	// NonActiveStreamIds represents a map, where key is the request subject where consumer sent
	// request for opening a stream, and value is the list of streamIDs which should no longer be
	// active.
	NonActiveStreamIds map[string][]string
}

// CloseError represents a close message.
type CloseError struct {
	// Code is defined in RFC 6455, section 11.7.
	Code int
	// Text is the optional text payload.
	Text string
}

// CloseError implements the error interface.
func (e *CloseError) Error() string {
	s := []byte("nats stream: close ")
	s = strconv.AppendInt(s, int64(e.Code), 10) //nolint:gomnd
	switch e.Code {
	case CloseNormalClosure:
		s = append(s, " (normal)"...)
	case CloseGoingAway:
		s = append(s, " (going away)"...)
	case CloseUnsupportedData:
		s = append(s, " (unsupported data)"...)
	case CloseAbnormalClosure:
		s = append(s, " (abnormal closure)"...)
	case CloseBadRequest:
		s = append(s, " (bad request)"...)
	case CloseInternalServerErr:
		s = append(s, " (internal server error)"...)
	}
	if e.Text != "" {
		s = append(s, ": "...)
		s = append(s, e.Text...)
	}
	return string(s)
}
