package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type JobEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Node string      `json:"node"`
	Job  string      `json:"job"`
}

var Stdout = struct{ io.Writer }{os.Stdout}
var Stderr = struct{ io.Writer }{os.Stderr}

var nodeIDFieldName = "NodeID"

func init() { //nolint:gochecknoinits // init with zerolog is idiomatic
	zerolog.TimeFieldFormat = time.RFC3339Nano
	logLevelString := strings.ToLower(os.Getenv("LOG_LEVEL"))
	logTypeString := strings.ToLower(os.Getenv("LOG_TYPE"))

	switch {
	case logLevelString == "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case logLevelString == "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case logLevelString == "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case logLevelString == "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case logLevelString == "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	textWriter := zerolog.ConsoleWriter{Out: Stderr, TimeFormat: "15:04:05.999 |", NoColor: false, PartsOrder: []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.MessageFieldName}}

	// TODO: figure out a way to show the custom fields at the beginning of the log line rather than at the end.
	//  Adding the fields to the parts section didn't help as it just printed the fields twice.
	textWriter.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("[%s:", i)
	}

	textWriter.FormatFieldValue = func(i interface{}) string {
		// don't print nil in case field value wasn't preset. e.g. no nodeID
		if i == nil {
			i = ""
		}
		return fmt.Sprintf("%s]", i)
	}

	zerolog.CallerMarshalFunc = func(file string, line int) string {
		short := file

		separatorCount := 2
		countedSeparators := 0

		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				countedSeparators += 1
				if countedSeparators >= separatorCount {
					short = file[i+1:]
					break
				}
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}

	// we default to text output
	var useLogWriter io.Writer = textWriter

	if logTypeString == "json" {
		// we just want json
		useLogWriter = os.Stdout
	} else if logTypeString == "combined" {
		// we just want json and text and events
		useLogWriter = zerolog.MultiLevelWriter(textWriter, os.Stdout)
	} else if logTypeString == "event" {
		// we just want events
		useLogWriter = io.Discard
	}

	log.Logger = zerolog.New(useLogWriter).With().Timestamp().Caller().Logger()
}

func LoggerWithRuntimeInfo(runtimeInfo string) zerolog.Logger {
	return log.With().Str("R", runtimeInfo).Logger()
}

func LoggerWithNodeID(nodeID string) zerolog.Logger {
	if len(nodeID) > 8 { //nolint:gomnd // 8 is a magic number
		nodeID = nodeID[:8]
	}
	return log.With().Str(nodeIDFieldName, nodeID).Logger()
}

// return a context with nodeID is added to the logging context.
func ContextWithNodeIDLogger(ctx context.Context, nodeID string) context.Context {
	l := LoggerWithNodeID(nodeID)
	return l.WithContext(ctx)
}

func LoggerTestLogger(logBuffer *bytes.Buffer) zerolog.Logger {
	return zerolog.New(zerolog.MultiLevelWriter(io.MultiWriter(logBuffer, os.Stdout)))
}
