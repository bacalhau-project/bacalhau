package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/pkgerrors"

	"github.com/filecoin-project/bacalhau/pkg/model"
	ipfslog2 "github.com/ipfs/go-log/v2"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var nodeIDFieldName = "NodeID"

func init() { //nolint:gochecknoinits // init with zerolog is idiomatic
	configureLogging()
}

type tTesting interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Helper()
	Cleanup(f func())
}

// ConfigureTestLogging allows logs to be associated with individual tests
func ConfigureTestLogging(t tTesting) {
	oldLogger := log.Logger
	oldContextLogger := zerolog.DefaultContextLogger
	oldCallerMarshaller := zerolog.CallerMarshalFunc
	configureLogging(zerolog.ConsoleTestWriter(t))

	// When run as a test, the `file` will be relative to the directory containing the tests, so will be like
	// `../../compute/endpoint.go`. This could be changed to rebuild the full package from the relative one by
	// converting the relative path into an absolute one and then working out what to trim off, but this will do for the
	// moment. If https://github.com/golang/go/issues/33976 is fixed, then we could just rely on the `debug.BuildInfo`.
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
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
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.DefaultContextLogger = oldContextLogger
		zerolog.CallerMarshalFunc = oldCallerMarshaller
		configureIpfsLogging(log.Logger)
	})
}

func configureLogging(loggingOptions ...func(w *zerolog.ConsoleWriter)) {
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

	isTerminal := isatty.IsTerminal(os.Stdout.Fd())

	defaultLogging := func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.NoColor = !isTerminal
		w.TimeFormat = "15:04:05.999 |"
		w.PartsOrder = []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.CallerFieldName,
			zerolog.MessageFieldName,
		}

		// TODO: figure out a way to show the custom fields at the beginning of the log line rather than at the end.
		//  Adding the fields to the parts section didn't help as it just printed the fields twice.
		w.FormatFieldName = func(i interface{}) string {
			return fmt.Sprintf("[%s:", i)
		}

		w.FormatFieldValue = func(i interface{}) string {
			// don't print nil in case field value wasn't preset. e.g. no nodeID
			if i == nil {
				i = ""
			}
			return fmt.Sprintf("%s]", i)
		}
	}

	loggingOptions = append([]func(w *zerolog.ConsoleWriter){defaultLogging}, loggingOptions...)

	textWriter := zerolog.NewConsoleWriter(loggingOptions...)

	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Path != "" {
		zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
			file = strings.TrimPrefix(file, info.Main.Path+"/")
			return file + ":" + strconv.Itoa(line)
		}
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

	log.Logger = zerolog.New(useLogWriter).With().Timestamp().Caller().Stack().Logger()
	// While the normal flow will use ContextWithNodeIDLogger, this won't be so for tests.
	// Tests will use the DefaultContextLogger instead
	zerolog.DefaultContextLogger = &log.Logger

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	configureIpfsLogging(log.Logger)
}

func loggerWithNodeID(nodeID string) zerolog.Logger {
	if len(nodeID) > 8 { //nolint:gomnd // 8 is a magic number
		nodeID = nodeID[:model.ShortIDLength]
	}
	return log.With().Str(nodeIDFieldName, nodeID).Logger()
}

// ContextWithNodeIDLogger will return a context with nodeID is added to the logging context.
func ContextWithNodeIDLogger(ctx context.Context, nodeID string) context.Context {
	l := loggerWithNodeID(nodeID)
	return l.WithContext(ctx)
}

type zerologWriteSyncer struct {
	l zerolog.Logger
}

var _ zapcore.WriteSyncer = (*zerologWriteSyncer)(nil)

func (z *zerologWriteSyncer) Write(b []byte) (int, error) {
	z.l.Log().CallerSkipFrame(5).Msg(string(b)) //nolint:gomnd
	return len(b), nil
}

func (z *zerologWriteSyncer) Sync() error {
	return nil
}

func configureIpfsLogging(l zerolog.Logger) {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {}
	encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
	encCfg.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {}
	encCfg.ConsoleSeparator = " "
	encoder := zapcore.NewConsoleEncoder(encCfg)

	core := zapcore.NewCore(encoder, &zerologWriteSyncer{l: l}, zap.NewAtomicLevelAt(zapcore.DebugLevel))

	ipfslog2.SetPrimaryCore(core)
}
