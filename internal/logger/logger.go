package logger

import (
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLog *zap.Logger
var sugar *zap.SugaredLogger

func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15h04m05.00"))
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func init() {
	var err error
	config := zap.NewProductionConfig()

	var l zapcore.Level

	logLevelString := strings.ToLower(os.Getenv("LOG_LEVEL"))
	jsonLogging, err := strconv.ParseBool(os.Getenv("JSON_LOGGING"))

	if err != nil {
		jsonLogging = false
	}

	switch {
	case logLevelString == "debug":
		l = zapcore.DebugLevel
	case logLevelString == "error":
		l = zapcore.ErrorLevel
	case logLevelString == "warn":
		l = zapcore.WarnLevel
	case logLevelString == "fatal":
		l = zapcore.FatalLevel
	default:
		l = zapcore.InfoLevel
	}

	config.Level.SetLevel(l)

	enccoderConfig := zap.NewProductionEncoderConfig()
	zapcore.TimeEncoderOfLayout("Jan _2 15:04:05.000000000")

	if !jsonLogging {
		config.Encoding = "console"
		enccoderConfig.MessageKey = "message"
		enccoderConfig.LevelKey = "level"
		enccoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		enccoderConfig.TimeKey = "time"
		enccoderConfig.EncodeTime = CustomTimeEncoder
		enccoderConfig.CallerKey = "caller"
		enccoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	enccoderConfig.StacktraceKey = "" // to hide stacktrace info
	config.EncoderConfig = enccoderConfig

	zapLog, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
	sugar = zapLog.Sugar()
	sugar.Debugf("Log level from LOG_LEVEL_ENV_VAR: %s\nZap log level: %s", logLevelString, l)
}

func Info(args ...interface{}) {
	sugar.Info(args)
}

func Infof(message string, args ...interface{}) {
	sugar.Infof(message, args...)
}

func Debug(args ...interface{}) {
	sugar.Debug(args)
}

func Debugf(message string, args ...interface{}) {
	sugar.Debugf(message, args...)
}

func Warn(args ...interface{}) {
	sugar.Warn(args)
}

func Warnf(message string, args ...interface{}) {
	sugar.Warnf(message, args...)
}

func Error(args ...interface{}) {
	sugar.Error(args)
}

func Errorf(message string, args ...interface{}) {
	sugar.Errorf(message, args...)
}

func Fatal(args ...interface{}) {
	sugar.Fatal(args)
}
func Fatalf(message string, args ...interface{}) {
	sugar.Fatalf(message, args...)
}
