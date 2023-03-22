package wasmlogs

import "time"

type Message struct {
	Stream    string `json:"stream"`
	Data      []byte `json:"data"`
	Timestamp int64  `json:"ts"`
}

func NewMessage(stream string, data []byte) *Message {
	msg := &Message{
		Stream:    stream,
		Timestamp: time.Now().Unix(),
	}
	msg.Data = append([]byte(nil), data...)
	return msg
}
