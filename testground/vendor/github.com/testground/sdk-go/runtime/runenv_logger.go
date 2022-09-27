package runtime

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (re *RunEnv) initLogger() {
	level := zap.NewAtomicLevel()

	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if err := level.UnmarshalText([]byte(lvl)); err != nil {
			defer func() {
				// once the logger is defined...
				if re.logger != nil {
					re.logger.Sugar().Errorf("failed to decode log level '%q': %s", lvl, err)
				}
			}()
		}
	} else {
		level.SetLevel(zapcore.InfoLevel)
	}

	paths := []string{"stdout"}
	if re.TestOutputsPath != "" {
		paths = append(paths, filepath.Join(re.TestOutputsPath, "run.out"))
	}

	cfg := zap.Config{
		Development:       false,
		Level:             level,
		DisableCaller:     true,
		DisableStacktrace: true,
		OutputPaths:       paths,
		Encoding:          "json",
		InitialFields: map[string]interface{}{
			"run_id":   re.TestRun,
			"group_id": re.TestGroupID,
		},
	}

	enc := zap.NewProductionEncoderConfig()
	enc.LevelKey, enc.NameKey = "", ""
	enc.EncodeTime = zapcore.EpochNanosTimeEncoder
	cfg.EncoderConfig = enc

	var err error
	re.logger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
}
