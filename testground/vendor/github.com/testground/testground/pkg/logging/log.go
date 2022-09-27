package logging

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ()

var (
	encConfig zapcore.EncoderConfig
	encoder   zapcore.Encoder

	stdout zapcore.WriteSyncer
	stderr zapcore.WriteSyncer

	level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	terminal = true

	global Logging
)

func init() {
	encConfig = zap.NewDevelopmentEncoderConfig()
	encConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encConfig.EncodeCaller = nil
	encConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.StampMicro))
	}

	encoder = zapcore.NewConsoleEncoder(encConfig)

	sout, closer, err := zap.Open("stdout")
	if err != nil {
		closer()
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	serr, closer, err := zap.Open("stderr")
	if err != nil {
		closer()
		panic(fmt.Errorf("failed to initialize logger: %w", err))
	}

	stdout = sout
	stderr = serr

	global = NewLogging(NewLogger())
}

// IsTerminal returns whether we're running in terminal mode.
func IsTerminal() bool {
	return terminal
}

// SetLevel adjusts the level of the loggers.
func SetLevel(l zapcore.Level) {
	level.SetLevel(l)
}

// NewLogger returns a logger that outputs to stdout AND any extra WriteSyncers
// that have been passed in.
func NewLogger(extraWs ...zapcore.WriteSyncer) *zap.Logger {
	wss := append([]zapcore.WriteSyncer{stdout}, extraWs...)
	ws := zapcore.NewMultiWriteSyncer(wss...)

	core := zapcore.NewCore(encoder, ws, level)
	return zap.New(core, zap.ErrorOutput(stderr))
}

// L returns the global raw logger.
func L() *zap.Logger {
	return global.L()
}

// S returns the global sugared logger.
func S() *zap.SugaredLogger {
	return global.S()
}

func Encoder() zapcore.Encoder {
	return encoder
}

// Logging is a simple mixin for types with attached loggers.
type Logging struct {
	logger  *zap.Logger
	sugared *zap.SugaredLogger
}

// NewLogging is a convenience method for constructing a Logging.
func NewLogging(logger *zap.Logger) Logging {
	return Logging{
		logger:  logger,
		sugared: logger.Sugar(),
	}
}

// L returns the raw logger.
func (l *Logging) L() *zap.Logger {
	return l.logger
}

// S returns the sugared logger.
func (l *Logging) S() *zap.SugaredLogger {
	return l.sugared
}
