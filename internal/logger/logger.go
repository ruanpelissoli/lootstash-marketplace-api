package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Log is the global logger instance
var Log *slog.Logger

// Init initializes the global logger with the specified configuration
func Init(level string, jsonOutput bool) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     parseLevel(level),
		AddSource: true,
	}

	if jsonOutput {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)
}

// parseLevel converts a string log level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Context keys for request tracing
type ctxKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ctxKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey ctxKey = "user_id"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// FromContext returns a logger with context values (request_id, user_id) attached
func FromContext(ctx context.Context) *slog.Logger {
	logger := Log
	if logger == nil {
		logger = slog.Default()
	}

	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		logger = logger.With("request_id", reqID)
	}

	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// GetLevel returns the current log level as a string
func GetLevel() string {
	if Log == nil {
		return "info"
	}
	if Log.Handler().Enabled(context.Background(), slog.LevelDebug) {
		return "debug"
	}
	if Log.Handler().Enabled(context.Background(), slog.LevelInfo) {
		return "info"
	}
	if Log.Handler().Enabled(context.Background(), slog.LevelWarn) {
		return "warn"
	}
	return "error"
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	if Log == nil {
		return false
	}
	return Log.Handler().Enabled(context.Background(), slog.LevelDebug)
}
