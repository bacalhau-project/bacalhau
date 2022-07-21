package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/storage/util"
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

func init() { // nolint:gochecknoinits // init with zerolog is idiomatic
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
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

	textWriter := zerolog.ConsoleWriter{Out: Stdout, TimeFormat: "[0607]", NoColor: false, PartsOrder: []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.MessageFieldName}}

	textWriter.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	textWriter.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	textWriter.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s", i))
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

func LoggerWithNodeAndJobInfo(nodeID, jobID string) zerolog.Logger {
	return log.With().Str("N", nodeID).Str("J", jobID).Logger()
}

func LoggerTestLogger(logBuffer *bytes.Buffer) zerolog.Logger {
	return zerolog.New(zerolog.MultiLevelWriter(io.MultiWriter(logBuffer, os.Stdout)))
}

func LogJobEvent(event JobEvent) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in LogJobEvent. Error:\n", r)
		}
	}()

	event.Node = event.Node[:8]
	event.Job = event.Job[:8]
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	if os.Getenv("LOG_EVENT_FILE") != "" {
		f, err := os.OpenFile(os.Getenv("LOG_EVENT_FILE"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, util.OS_ALL_R|util.OS_USER_RW)
		if err != nil {
			return
		}
		defer f.Close()
		if _, err := f.WriteString(fmt.Sprintf("%s\n", string(eventBytes))); err != nil {
			return
		}
	}

	logType := strings.ToLower(os.Getenv("LOG_TYPE"))
	if logType != "event" && logType != "combined" {
		return
	}

	fmt.Println(string(eventBytes))
}
