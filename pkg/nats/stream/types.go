package stream

import "strconv"

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

type ConnectionDetails struct {
	// ConnId is the connection id of the consumer streaming client originating the request.
	ConnId string `json:"connId"`
	// StreamId is the id of the stream being created.
	StreamId string `json:"streamId"`
	// HeartBeatSub is the heart beat subject where the producer client will send its heart beat.
	HeartBeatRequestSub string `json:"heartBeatRequestSub"`
}

// HeartBeatResponse represents a list of stream ids that are Consumer Client is
// still interested in.
type HeartBeatResponse struct {
	StreamIds []string
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
