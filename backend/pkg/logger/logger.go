package logger

import (
	"context"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	TraceIDKey  contextKey = "trace_id"
	PlayerIDKey contextKey = "player_id"
)

var (
	L *zap.Logger        = zap.NewNop()
	S *zap.SugaredLogger = zap.NewNop().Sugar()
)

// WithTraceID embeds a trace ID into the given context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// TraceIDFromContext retrieves a trace ID from context, or returns empty string.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(TraceIDKey).(string); ok {
		return val
	}
	return ""
}

// WithPlayerID embeds a player ID into the given context.
func WithPlayerID(ctx context.Context, playerID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, PlayerIDKey, playerID)
}

// PlayerIDFromContext retrieves a player ID from context, or returns empty string.
func PlayerIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val, ok := ctx.Value(PlayerIDKey).(string); ok {
		return val
	}
	return ""
}

// FromContext returns a SugaredLogger contextualized with trace_id and player_id if present.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil || S == nil {
		return S
	}
	var fields []interface{}
	if tid := TraceIDFromContext(ctx); tid != "" {
		fields = append(fields, "trace_id", tid)
	}
	if pid := PlayerIDFromContext(ctx); pid != "" {
		fields = append(fields, "player_id", pid)
	}
	if len(fields) == 0 {
		return S
	}
	return S.With(fields...)
}

// LFromContext returns a Zap Logger contextualized with trace_id and player_id if present.
func LFromContext(ctx context.Context) *zap.Logger {
	if ctx == nil || L == nil {
		return L
	}
	var fields []zap.Field
	if tid := TraceIDFromContext(ctx); tid != "" {
		fields = append(fields, zap.String("trace_id", tid))
	}
	if pid := PlayerIDFromContext(ctx); pid != "" {
		fields = append(fields, zap.String("player_id", pid))
	}
	if len(fields) == 0 {
		return L
	}
	return L.With(fields...)
}

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
