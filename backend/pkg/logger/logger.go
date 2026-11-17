package logger

import (
	"context"
	"errors"
	"fmt"
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
	L *zap.Logger    = zap.NewNop()
	S *SugaredLogger = newSugaredLogger(zap.NewNop())
)

// SugaredLogger enriches every sugared error event with its wrapped cause chain.
type SugaredLogger struct {
	*zap.SugaredLogger
	errorLogger *zap.SugaredLogger
}

func newSugaredLogger(logger *zap.Logger) *SugaredLogger {
	return &SugaredLogger{
		SugaredLogger: logger.Sugar(),
		errorLogger:   logger.WithOptions(zap.AddCallerSkip(1)).Sugar(),
	}
}

func (l *SugaredLogger) Error(args ...interface{}) {
	l.logError(fmt.Sprint(args...), args...)
}

func (l *SugaredLogger) Errorf(template string, args ...interface{}) {
	l.logError(fmt.Sprintf(template, args...), args...)
}

func (l *SugaredLogger) Errorln(args ...interface{}) {
	l.logError(strings.TrimSuffix(fmt.Sprintln(args...), "\n"), args...)
}

func (l *SugaredLogger) Warn(args ...interface{}) {
	l.logWarning(fmt.Sprint(args...), args...)
}

func (l *SugaredLogger) Warnf(template string, args ...interface{}) {
	l.logWarning(fmt.Sprintf(template, args...), args...)
}

func (l *SugaredLogger) Warnln(args ...interface{}) {
	l.logWarning(strings.TrimSuffix(fmt.Sprintln(args...), "\n"), args...)
}

func (l *SugaredLogger) Errorw(message string, keysAndValues ...interface{}) {
	chain, root := errorFieldsFromValues(keysAndValues)
	if len(chain) > 0 {
		keysAndValues = append(keysAndValues, "error_chain", chain, "root_cause", root)
	}
	l.errorLogger.Errorw(message, keysAndValues...)
}

func (l *SugaredLogger) Warnw(message string, keysAndValues ...interface{}) {
	chain, root := errorFieldsFromValues(keysAndValues)
	if len(chain) > 0 {
		keysAndValues = append(keysAndValues, "error_chain", chain, "root_cause", root)
	}
	l.errorLogger.Warnw(message, keysAndValues...)
}

func (l *SugaredLogger) With(keysAndValues ...interface{}) *SugaredLogger {
	return &SugaredLogger{
		SugaredLogger: l.SugaredLogger.With(keysAndValues...),
		errorLogger:   l.errorLogger.With(keysAndValues...),
	}
}

func (l *SugaredLogger) logError(message string, values ...interface{}) {
	l.logAtErrorLevel(message, zapcore.ErrorLevel, values...)
}

func (l *SugaredLogger) logWarning(message string, values ...interface{}) {
	l.logAtErrorLevel(message, zapcore.WarnLevel, values...)
}

func (l *SugaredLogger) logAtErrorLevel(message string, level zapcore.Level, values ...interface{}) {
	chain, root := errorFieldsFromValues(values)
	if len(chain) == 0 {
		if level == zapcore.WarnLevel {
			l.errorLogger.Desugar().Warn(message)
		} else {
			l.errorLogger.Desugar().Error(message)
		}
		return
	}
	fields := []zap.Field{zap.Strings("error_chain", chain), zap.String("root_cause", root)}
	if level == zapcore.WarnLevel {
		l.errorLogger.Desugar().Warn(message, fields...)
	} else {
		l.errorLogger.Desugar().Error(message, fields...)
	}
}

func errorFieldsFromValues(values []interface{}) ([]string, string) {
	for _, value := range values {
		if err, ok := value.(error); ok && err != nil {
			chain, root := ErrorDetails(err)
			return chain, root
		}
	}
	return nil, ""
}

// ErrorDetails returns the complete wrapped error chain and its deepest causes.
func ErrorDetails(err error) ([]string, string) {
	if err == nil {
		return nil, ""
	}
	var chain []string
	var roots []string
	var walk func(error)
	walk = func(current error) {
		if current == nil {
			return
		}
		chain = append(chain, current.Error())
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			children := joined.Unwrap()
			if len(children) == 0 {
				roots = append(roots, current.Error())
				return
			}
			for _, child := range children {
				walk(child)
			}
			return
		}
		if cause := errors.Unwrap(current); cause != nil {
			walk(cause)
			return
		}
		roots = append(roots, current.Error())
	}
	walk(err)
	return chain, strings.Join(roots, "; ")
}

// ErrorFields returns stable structured fields for an error and its root cause.
func ErrorFields(err error) []zap.Field {
	if err == nil {
		return nil
	}
	chain, root := ErrorDetails(err)
	return []zap.Field{zap.Strings("error_chain", chain), zap.String("root_cause", root)}
}

// LogError writes one structured error event with contextual IDs and root cause.
func LogError(ctx context.Context, message string, err error, fields ...zap.Field) {
	if err == nil {
		return
	}
	fields = append(fields, ErrorFields(err)...)
	LFromContext(ctx).Error(message, fields...)
}

type errorDetailsCore struct{ zapcore.Core }

func (c errorDetailsCore) With(fields []zap.Field) zapcore.Core {
	return errorDetailsCore{Core: c.Core.With(fields)}
}

func (c errorDetailsCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c errorDetailsCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	var errs []error
	hasChain := false
	for _, field := range fields {
		if field.Key == "error_chain" {
			hasChain = true
		}
		if field.Type == zapcore.ErrorType || field.Type == zapcore.ReflectType {
			if err, ok := field.Interface.(error); ok && err != nil {
				errs = append(errs, err)
			}
		}
	}
	if entry.Level >= zapcore.WarnLevel && len(errs) > 0 && !hasChain {
		var chains []string
		var roots []string
		for _, err := range errs {
			chain, root := ErrorDetails(err)
			chains = append(chains, chain...)
			if root != "" {
				roots = append(roots, root)
			}
		}
		fields = append(fields, zap.Strings("error_chain", chains), zap.String("root_cause", strings.Join(roots, "; ")))
	}
	return c.Core.Write(entry, fields)
}

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
	if S == nil {
		return nil
	}
	if ctx == nil {
		return S.SugaredLogger
	}
	var fields []interface{}
	if tid := TraceIDFromContext(ctx); tid != "" {
		fields = append(fields, "trace_id", tid)
	}
	if pid := PlayerIDFromContext(ctx); pid != "" {
		fields = append(fields, "player_id", pid)
	}
	if len(fields) == 0 {
		return S.SugaredLogger
	}
	return S.SugaredLogger.With(fields...)
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
	configuredLevel := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	var invalidLevel string
	if configuredLevel != "" {
		if err := level.UnmarshalText([]byte(strings.ToLower(configuredLevel))); err != nil {
			invalidLevel = configuredLevel
			level = zapcore.InfoLevel
		}
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

	core := errorDetailsCore{Core: zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)}

	logger := zap.New(core, zap.AddCaller())

	L = logger
	S = newSugaredLogger(logger)
	if invalidLevel != "" {
		S.Warnw("invalid LOG_LEVEL; using INFO", "configured_level", invalidLevel)
	}
	return nil
}
