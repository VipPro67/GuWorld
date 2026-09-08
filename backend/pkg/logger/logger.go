package logger

import (
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	L *zap.Logger        = zap.NewNop()
	S *zap.SugaredLogger = zap.NewNop().Sugar()
)

type ginWriter struct{}

func (g *ginWriter) Write(p []byte) (n int, err error) {
	S.Info(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

func GinWriter() io.Writer {
	return &ginWriter{}
}

func Sync() error {
	if L != nil {
		return L.Sync()
	}
	return nil
}

func Init() error {
	level := zapcore.InfoLevel
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		level = zapcore.DebugLevel
	case "INFO":
		level = zapcore.InfoLevel
	case "WARN":
		level = zapcore.WarnLevel
	case "ERROR":
		level = zapcore.ErrorLevel
	case "DPANIC":
		level = zapcore.DPanicLevel
	case "PANIC":
		level = zapcore.PanicLevel
	case "FATAL":
		level = zapcore.FatalLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Use JSON encoding when running in production/docker for structured logging,
	// or console encoding for development mode.
	var encoder zapcore.Encoder
	if strings.ToLower(os.Getenv("PROD_ENV")) == "production" || os.Getenv("JSON_LOGS") == "true" {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)

	logger := zap.New(core, zap.AddCaller())

	L = logger
	S = logger.Sugar()
	return nil
}
