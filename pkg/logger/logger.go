package logger

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Stdout = struct{ io.Writer }{os.Stdout}
var Stderr = struct{ io.Writer }{os.Stderr}

func Initialize() {
	// Needs no functionality, but need some function to create
}

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

	var useLogWriter io.Writer

	// we default to text output
	if logTypeString == "" {
		useLogWriter = textWriter
	} else if logTypeString == "json" {
		useLogWriter = os.Stdout
	} else if logTypeString == "combined" {
		useLogWriter = zerolog.MultiLevelWriter(textWriter, os.Stdout)
	}

	log.Logger = zerolog.New(useLogWriter).With().Timestamp().Caller().Logger()

}

func LoggerWithRuntimeInfo(runtimeInfo string) zerolog.Logger {
	return log.With().Str("R", runtimeInfo).Logger()
}

func LoggerWithNodeAndJobInfo(nodeId string, jobId string) zerolog.Logger {
	return log.With().Str("N", nodeId).Str("J", jobId).Logger()
}
