package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

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

func init() {
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

	//file, _ := ioutil.TempFile("tmp", "logs")

	textWriter := zerolog.ConsoleWriter{Out: Stdout, TimeFormat: "[0607]", NoColor: false, PartsOrder: []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.MessageFieldName}}

	// output.FormatLevel = func(i interface{}) string {
	// 	return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	// }
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

		seperator_count := 2
		counted_separators := 0

		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				counted_separators += 1
				if counted_separators >= seperator_count {
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
		useLogWriter = ioutil.Discard
	}

	log.Logger = zerolog.New(useLogWriter).With().Timestamp().Caller().Logger()

}

func LoggerWithRuntimeInfo(runtimeInfo string) zerolog.Logger {
	return log.With().Str("R", runtimeInfo).Logger()
}

func LoggerWithNodeAndJobInfo(nodeID string, jobId string) zerolog.Logger {
	return log.With().Str("N", nodeID).Str("J", jobId).Logger()
}

func LogJobEvent(event JobEvent) {
	event.Node = event.Node[:8]
	event.Job = event.Job[:8]
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return
	}

	if os.Getenv("LOG_EVENT_FILE") != "" {
		f, err := os.OpenFile(os.Getenv("LOG_EVENT_FILE"),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
