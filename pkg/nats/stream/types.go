package stream

import (
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
	ConnectionDetails ConnectionDetails `json:"connectionDetails"`
	Data              []byte            `json:"body"`
}

// ConnectionDetails represents the details of incoming stream connection.
type ConnectionDetails struct {
	// ConnID is the connection id of the consumer streaming client originating the request.
	ConnID string `json:"connId"`
	// StreamId is the id of the stream being created.
	StreamID string `json:"streamId"`
	// HeartBeatSub is the heart beat subject where the producer client will send its heart beat.
	HeartBeatRequestSub string `json:"heartBeatRequestSub"`
}

// StreamInfo represents information about the stream.
type StreamInfo struct {
	// ID is the identifier of the stream.
	ID string
	// RequestSub is the subject on which the request for this stream was sent.
	RequestSub string
	// CreatedAt represents the time the stream was created.
	CreatedAt time.Time
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
	// NonActiveStreamIds represent a list of stream ids which the consumer client is not
	// interested in.
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
