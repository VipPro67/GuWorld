package httperr

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
)

// AppError is a structured error carrying an HTTP status code, safe user message,
// root-cause error and optional extra details.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
	Details any    `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New creates a new *AppError with given HTTP status code, user message and root cause.
func New(code int, message string, internalErr error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     internalErr,
	}
}

// BadRequest constructs a 400 Bad Request AppError.
func BadRequest(message string, internalErr ...error) *AppError {
	var err error
	if len(internalErr) > 0 {
		err = internalErr[0]
	}
	return New(http.StatusBadRequest, message, err)
}

// Unauthorized constructs a 401 Unauthorized AppError.
func Unauthorized(message string, internalErr ...error) *AppError {
	var err error
	if len(internalErr) > 0 {
		err = internalErr[0]
	}
	return New(http.StatusUnauthorized, message, err)
}

// Forbidden constructs a 403 Forbidden AppError.
func Forbidden(message string, internalErr ...error) *AppError {
	var err error
	if len(internalErr) > 0 {
		err = internalErr[0]
	}
	return New(http.StatusForbidden, message, err)
}

// NotFound constructs a 404 Not Found AppError.
func NotFound(message string, internalErr ...error) *AppError {
	var err error
	if len(internalErr) > 0 {
		err = internalErr[0]
	}
	return New(http.StatusNotFound, message, err)
}

// Conflict constructs a 409 Conflict AppError.
func Conflict(message string, internalErr ...error) *AppError {
	var err error
	if len(internalErr) > 0 {
		err = internalErr[0]
	}
	return New(http.StatusConflict, message, err)
}

// Internal constructs a 500 Internal Server Error AppError.
func Internal(message string, internalErr error) *AppError {
	return New(http.StatusInternalServerError, message, internalErr)
}

func isDebugMode() bool {
	return os.Getenv("DEBUG_ERRORS") == "true" ||
		(strings.ToLower(os.Getenv("PROD_ENV")) != "production" && os.Getenv("GIN_MODE") != "release")
}

// Respond handles error emission across the handler seam:
// 1. Unwraps and classifies error (AppError vs arbitrary error).
// 2. Notifies Gin via c.Error(err) so monitor and middlewares capture the real error.
// 3. Emits structured contextual log to Zap (including trace_id, route, player_id).
// 4. Returns standardized sanitized JSON response: {"error": message, "trace_id": trace_id}.
func Respond(c *gin.Context, err error) {
	if c == nil || err == nil {
		return
	}

	code := http.StatusInternalServerError
	userMsg := "Lỗi xử lý hệ thống"
	var internalErr error = err
	var details any

	var appErr *AppError
	if errors.As(err, &appErr) && appErr != nil {
		if appErr.Code > 0 {
			code = appErr.Code
		}
		if appErr.Message != "" {
			userMsg = appErr.Message
		}
		if appErr.Err != nil {
			internalErr = appErr.Err
		}
		details = appErr.Details
	}

	// Register error with Gin context so c.Errors contains it for middlewares and monitors
	_ = c.Error(err)

	ctx := c.Request.Context()
	traceID := logger.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = c.GetString("trace_id")
	}

	playerID, _ := c.Get("player_id")

	// Structured logging with context
	logFields := []zap.Field{
		zap.String("trace_id", traceID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("route", c.FullPath()),
		zap.Int("status", code),
		zap.Any("player_id", playerID),
		zap.Error(internalErr),
	}

	if code >= 500 {
		logger.LFromContext(ctx).Error("http_handler_internal_error", logFields...)
	} else if code >= 400 {
		logger.LFromContext(ctx).Warn("http_handler_client_error", logFields...)
	}

	resp := gin.H{
		"error":    userMsg,
		"trace_id": traceID,
	}
	if details != nil {
		resp["details"] = details
	}
	if isDebugMode() && internalErr != nil {
		resp["debug"] = internalErr.Error()
	}

	c.JSON(code, resp)
}

// AbortWithError calls Respond and then aborts the context.
func AbortWithError(c *gin.Context, err error) {
	Respond(c, err)
	c.Abort()
}

// AbortBadRequest is a convenience helper for 400 Bad Request.
func AbortBadRequest(c *gin.Context, userMsg string, internalErr ...error) {
	AbortWithError(c, BadRequest(userMsg, internalErr...))
}

// AbortUnauthorized is a convenience helper for 401 Unauthorized.
func AbortUnauthorized(c *gin.Context, userMsg string, internalErr ...error) {
	AbortWithError(c, Unauthorized(userMsg, internalErr...))
}

// AbortForbidden is a convenience helper for 403 Forbidden.
func AbortForbidden(c *gin.Context, userMsg string, internalErr ...error) {
	AbortWithError(c, Forbidden(userMsg, internalErr...))
}

// AbortNotFound is a convenience helper for 404 Not Found.
func AbortNotFound(c *gin.Context, userMsg string, internalErr ...error) {
	AbortWithError(c, NotFound(userMsg, internalErr...))
}

// AbortInternal is a convenience helper for 500 Internal Server Error.
func AbortInternal(c *gin.Context, userMsg string, internalErr error) {
	AbortWithError(c, Internal(userMsg, internalErr))
}
