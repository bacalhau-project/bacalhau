package logger

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"go/build"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/pkgerrors"

	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/zap/zapcore"
)

type LogMode string

// Available logging modes
const (
	LogModeDefault LogMode = "default"
	LogModeJSON    LogMode = "json"
	LogModeCmd     LogMode = "cmd"
)

var (
	logMu sync.Mutex
)

func ParseLogMode(s string) (LogMode, error) {
	lm := []LogMode{LogModeDefault, LogModeJSON, LogModeCmd}
	for _, logMode := range lm {
		if strings.EqualFold(s, string(logMode)) {
			return logMode, nil
		}
	}
	return "", fmt.Errorf("%q is an invalid log-mode (valid modes: %q)", s, lm)
}

func ParseLogLevel(s string) (zerolog.Level, error) {
	l, err := zerolog.ParseLevel(s)
	if err != nil {
		return l, fmt.Errorf("%q is an invalid log-level", s)
	}
	return l, nil
}

var nodeIDFieldName = "NodeID"

func init() { //nolint:gochecknoinits
	// logging needs to be automatically configured when running as a test.
	// Buffer the log messages till logging has been configured when not running as a test, so they can be outputted
	// in the correct format.
	if strings.Contains(os.Args[0], "/_test/") ||
		strings.HasSuffix(os.Args[0], ".test") ||
		flag.Lookup("test.v") != nil ||
		flag.Lookup("test.run") != nil {
		ConfigureLoggingLevel(zerolog.DebugLevel)
		configureLogging(defaultLogging())
		return
	}

	// the default log level when not running a test is ERROR
	ConfigureLoggingLevel(zerolog.ErrorLevel)
	configureLogging(bufferLogs())
}

func ErrOrDebug(err error) zerolog.Level {
	if err == nil {
		return zerolog.DebugLevel
	} else {
		return zerolog.ErrorLevel
	}
}

type tTesting interface {
	zerolog.TestingLog
	Cleanup(f func())
}

// ConfigureTestLogging allows logs to be associated with individual tests
func ConfigureTestLogging(t tTesting) {
	oldLogger := log.Logger
	oldContextLogger := zerolog.DefaultContextLogger
	ConfigureLoggingLevel(zerolog.DebugLevel)
	configureLogging(zerolog.NewConsoleWriter(zerolog.ConsoleTestWriter(t), defaultLogFormat))
	t.Cleanup(func() {
		log.Logger = oldLogger
		zerolog.DefaultContextLogger = oldContextLogger
	})
}

func ParseAndConfigureLogging(modeStr, levelStr string) error {
	mode, err := ParseLogMode(modeStr)
	if err != nil {
		return err
	}
	level, err := ParseLogLevel(levelStr)
	if err != nil {
		return err
	}

	ConfigureLogging(mode, level)
	return nil
}

func ConfigureLogging(mode LogMode, level zerolog.Level) {
	// set global log level before configuring logging as it is used in the configuration
	ConfigureLoggingLevel(level)

	var logWriter io.Writer
	switch mode {
	case LogModeDefault:
		logWriter = defaultLogging()
	case LogModeJSON:
		logWriter = jsonLogging()
	case LogModeCmd:
		logWriter = clientLogging()
	default:
		logWriter = defaultLogging()
	}

	configureLogging(logWriter)
	LogBufferedLogs(logWriter)
}

func ParseAndConfigureLoggingLevel(level string) error {
	l, err := ParseLogLevel(level)
	if err != nil {
		return err
	}
	ConfigureLoggingLevel(l)
	return nil
}

func ConfigureLoggingLevel(level zerolog.Level) {
	logMu.Lock()
	defer logMu.Unlock()
	zerolog.SetGlobalLevel(level)
}

func configureLogging(logWriter io.Writer) {
	logMu.Lock()
	defer logMu.Unlock()

	zerolog.TimeFieldFormat = time.RFC3339Nano

	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Path != "" {
		// Branch that'll be used when the binary is run, as it is built as a Go module
		zerolog.CallerMarshalFunc = marshalCaller(info.Main.Path)
	} else {
		// Branch typically used when running under test as build info isn't populated
		// https://github.com/golang/go/issues/33976
		dir := findRepositoryRoot()
		if dir != "" {
			zerolog.CallerMarshalFunc = marshalCaller(dir)
		}
	}

	log.Logger = zerolog.New(logWriter).With().Timestamp().Caller().Stack().Logger()
	// While the normal flow will use ContextWithNodeIDLogger, this won't be so for tests.
	// Tests will use the DefaultContextLogger instead
	zerolog.DefaultContextLogger = &log.Logger

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func jsonLogging() io.Writer {
	return os.Stdout
}

func defaultLogging() io.Writer {
	return zerolog.NewConsoleWriter(defaultLogFormat)
}

func defaultLogFormat(w *zerolog.ConsoleWriter) {
	isTerminal := isatty.IsTerminal(os.Stdout.Fd())
	w.Out = os.Stdout
	w.NoColor = !isTerminal

	// Get the current log level to determine format
	level := zerolog.GlobalLevel()
	isDebug := level <= zerolog.DebugLevel

	if isDebug {
		// Debug mode - show detailed information
		w.TimeFormat = "15:04:05.999 |"
		w.PartsOrder = []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.CallerFieldName,
			zerolog.MessageFieldName,
		}
	} else {
		// Normal mode - simplified, user-friendly format
		w.TimeFormat = "15:04:05 |"
		w.PartsOrder = []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.MessageFieldName,
		}
	}
}

func clientLogging() io.Writer {
	return zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		defaultLogFormat(w)
		w.PartsOrder = []string{zerolog.MessageFieldName}
	})
}

func loggerWithNodeID(nodeID string) zerolog.Logger {
	return log.With().Str(nodeIDFieldName, idgen.ShortNodeID(nodeID)).Logger()
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
	z.l.Log().CallerSkipFrame(5).Msg(string(b)) //nolint:mnd
	return len(b), nil
}

func (z *zerologWriteSyncer) Sync() error {
	return nil
}

func LogStream(ctx context.Context, r io.Reader) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		log.Ctx(ctx).Debug().Msg(s.Text())
	}
	if s.Err() != nil {
		log.Ctx(ctx).Error().Err(s.Err()).Msg("error consuming log")
	}
}

func findRepositoryRoot() string {
	dir, _ := os.Getwd()
	for {
		_, err := os.Stat(filepath.Join(dir, "go.mod"))
		if os.IsNotExist(err) {
			parentDir := filepath.Dir(dir)
			if dir == parentDir {
				return ""
			}
			dir = parentDir
			continue
		}
		return filepath.ToSlash(dir)
	}
}

func marshalCaller(prefix string) func(uintptr, string, int) string {
	goPath := build.Default.GOPATH
	// `file` will always use '/', even on Windows
	goPath = fmt.Sprintf("%s/%s/%s/", filepath.ToSlash(goPath), "pkg", "mod")
	return func(_ uintptr, file string, line int) string {
		if strings.HasPrefix(file, prefix) {
			file = strings.TrimPrefix(file, prefix+"/")
		} else {
			file = strings.TrimPrefix(file, goPath)
		}
		return file + ":" + strconv.Itoa(line)
	}
}

var _ zapcore.Core = &zerologZapCore{}

type zerologZapCore struct {
	l zerolog.Logger
}

func (z *zerologZapCore) Enabled(level zapcore.Level) bool {
	zerologLevel := marshalZapCoreLogLevel(level)

	return z.l.GetLevel() <= zerologLevel
}

func (z *zerologZapCore) With(fields []zapcore.Field) zapcore.Core {
	logCtx := marshalZapCoreFields(fields, z.l.With())

	return &zerologZapCore{logCtx.Logger()}
}

func (z *zerologZapCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if z.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, z)
	}

	return checkedEntry
}

// zapCoreCallDepth is how far zerologZapCore.Write is down the stack from someone calling `log.Error`
const zapCoreCallDepth = 4

func (z *zerologZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	e := z.l.
		WithLevel(marshalZapCoreLogLevel(entry.Level)).
		CallerSkipFrame(zapCoreCallDepth).
		Str("logger-name", entry.LoggerName)

	e = marshalZapCoreFields(fields, e)

	e.Msg(entry.Message)

	return nil
}

func (z *zerologZapCore) Sync() error {
	return nil
}

func marshalZapCoreLogLevel(level zapcore.Level) zerolog.Level {
	switch level {
	case zapcore.DebugLevel:
		return zerolog.DebugLevel
	case zapcore.InfoLevel:
		return zerolog.InfoLevel
	case zapcore.WarnLevel:
		return zerolog.WarnLevel
	case zapcore.ErrorLevel:
		return zerolog.ErrorLevel
	}

	return zerolog.PanicLevel
}

//nolint:gosec,gocyclo // Field marshaling requires handling many zap field types
func marshalZapCoreFields[T zerologFields[T]](fields []zapcore.Field, handler T) T {
	keyPrefix := ""

	for _, f := range fields {
		key := keyPrefix + f.Key
		switch f.Type {
		case zapcore.BinaryType:
			handler = handler.Bytes(key, f.Interface.([]byte))
		case zapcore.BoolType:
			handler = handler.Bool(key, f.Integer == 1)
		case zapcore.DurationType:
			handler = handler.Dur(key, time.Duration(f.Integer))
		case zapcore.Float64Type:
			handler = handler.Float64(key, math.Float64frombits(uint64(f.Integer)))
		case zapcore.Float32Type:
			handler = handler.Float32(key, math.Float32frombits(uint32(f.Integer)))
		case zapcore.Int64Type:
			handler = handler.Int64(key, f.Integer)
		case zapcore.Int32Type:
			handler = handler.Int32(key, int32(f.Integer))
		case zapcore.Int16Type:
			handler = handler.Int16(key, int16(f.Integer))
		case zapcore.Int8Type:
			handler = handler.Int8(key, int8(f.Integer))
		case zapcore.StringType:
			handler = handler.Str(key, f.String)
		case zapcore.TimeType:
			t := time.Unix(0, f.Integer)
			if f.Interface != nil {
				t = t.In(f.Interface.(*time.Location))
			}
			handler = handler.Time(key, t)
		case zapcore.TimeFullType:
			handler = handler.Time(key, f.Interface.(time.Time))
		case zapcore.Uint64Type:
			handler = handler.Uint64(key, uint64(f.Integer))
		case zapcore.Uint32Type:
			handler = handler.Uint32(key, uint32(f.Integer))
		case zapcore.Uint16Type:
			handler = handler.Uint16(key, uint16(f.Integer))
		case zapcore.Uint8Type:
			handler = handler.Uint8(key, uint8(f.Integer))
		case zapcore.NamespaceType:
			keyPrefix = f.Key
		case zapcore.StringerType:
			handler = handler.Stringer(key, f.Interface.(fmt.Stringer))
		case zapcore.ErrorType:
			handler = handler.AnErr(key, f.Interface.(error))
		case zapcore.SkipType:
		default:
			handler = handler.Interface(key, f.Interface)
		}
	}

	return handler
}

type zerologFields[T any] interface {
	Bytes(key string, val []byte) T
	Bool(key string, b bool) T
	Dur(key string, d time.Duration) T
	Float64(key string, f float64) T
	Float32(key string, f float32) T
	Int64(key string, i int64) T
	Int32(key string, i int32) T
	Int16(key string, i int16) T
	Int8(key string, i int8) T
	Str(key, val string) T
	Time(key string, t time.Time) T
	Uint64(key string, i uint64) T
	Uint32(key string, i uint32) T
	Uint16(key string, i uint16) T
	Uint8(key string, i uint8) T
	Stringer(key string, val fmt.Stringer) T
	AnErr(key string, err error) T
	Interface(key string, i interface{}) T
}
