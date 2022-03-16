package logger

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Initialize() {
	// Needs no functionality, but need some function to create
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevelString := strings.ToLower(os.Getenv("LOG_LEVEL"))

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

	output := zerolog.ConsoleWriter{Out: system.Stdout, TimeFormat: "[0607]", NoColor: false, PartsOrder: []string{
		zerolog.TimestampFieldName,
		zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.MessageFieldName}}

	// output.FormatLevel = func(i interface{}) string {
	// 	return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	// }
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
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

	log.Logger = zerolog.New(output).With().Timestamp().Caller().Logger()

}

func LoggerWithRuntimeInfo(runtimeInfo string) zerolog.Logger {
	return log.With().Str("R", runtimeInfo).Logger()
}

func LoggerWithNodeAndJobInfo(nodeId string, jobId string) zerolog.Logger {
	return log.With().Str("N", nodeId).Str("J", jobId).Logger()
}
