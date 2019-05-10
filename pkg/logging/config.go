package logging

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	LoggingConfig string
	LoggingLevel  map[string]zapcore.Level
}

const defaultZLC = `{
  "level": "info",
  "development": false,
  "outputPaths": ["stdout"],
  "errorOutputPaths": ["stderr"],
  "encoding": "json",
  "encoderConfig": {
    "timeKey": "ts",
    "levelKey": "level",
    "nameKey": "logger",
    "callerKey": "caller",
    "messageKey": "msg",
    "stacktraceKey": "stacktrace",
    "lineEnding": "",
    "levelEncoder": "",
    "timeEncoder": "iso8601",
    "durationEncoder": "",
    "callerEncoder": ""
  }
}`

func NewLogger(config *Config, name string, opts ...zap.Option) (*zap.SugaredLogger, zap.AtomicLevel) {
	logger, level := newLoggerFromConfig(config.LoggingConfig, config.LoggingLevel[name].String(), opts...)
	return logger.Named(name), level
}

func NewConfigFromMap(data map[string]string, components ...string) (*Config, error) {
	lc := &Config{}
	if zlc, ok := data["zap-logger-config"]; ok {
		lc.LoggingConfig = zlc
	} else {
		lc.LoggingConfig = defaultZLC
	}
	lc.LoggingLevel = make(map[string]zapcore.Level)
	for _, component := range components {
		if ll := data["loglevel."+component]; len(ll) > 0 {
			level, err := levelFromString(ll)
			if err != nil {
				return nil, err
			}
			lc.LoggingLevel[component] = *level
		} else {
			lc.LoggingLevel[component] = zapcore.InfoLevel
		}
	}
	return lc, nil
}

func newLoggerFromConfig(configJSON string, levelOverride string, opts ...zap.Option) (*zap.SugaredLogger, zap.AtomicLevel) {
	if len(configJSON) == 0 {
		panic("empty logging configuration")
		//return nil, zap.AtomicLevel{}, errors.New("empty logging configuration")
	}

	var loggingCfg zap.Config
	if err := json.Unmarshal([]byte(configJSON), &loggingCfg); err != nil {
		panic("Unmarshal configjson fail")
		//return nil, zap.AtomicLevel{}, err
	}

	if len(levelOverride) > 0 {
		if level, err := levelFromString(levelOverride); err == nil {
			loggingCfg.Level = zap.NewAtomicLevelAt(*level)
		}
	}

	logger, err := loggingCfg.Build(opts...)
	if err != nil {
		panic("build logger fail by loggingCfg")
		//return nil, zap.AtomicLevel{}, err
	}

	logger.Info("Successfully created the logger.", zap.String(JSONConfig, configJSON))
	logger.Sugar().Infof("Logging level set to %v", loggingCfg.Level)

	return logger.Sugar(), loggingCfg.Level

}

func levelFromString(level string) (*zapcore.Level, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid logging level: %v", level)
	}
	return &zapLevel, nil
}
