package wasmlogs

type LogStreamType int

const (
	LogStreamStdout = iota
	LogStreamStderr
)

func ToStreamName(t LogStreamType) string {
	if t == LogStreamStdout {
		return "stdout"
	} else if t == LogStreamStderr {
		return "stderr"
	}
	return ""
}

type LogMessage struct {
	Stream    string
	Data      []byte
	Timestamp int64
}
