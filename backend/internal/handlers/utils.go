package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

// APIResponse is the unified envelope for all HTTP responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError provides structured error details.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta holds optional pagination, timestamps, etc.
type Meta struct {
	Timestamp string `json:"timestamp,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// HandleError is a helper to reduce error handling boilerplate.
// Returns true if error was handled (response sent), false if err is nil.
// Usage:
//
//	if HandleError(c, err, "Failed to do something") {
//	    return
//	}
func HandleError(c *gin.Context, err error, msg string) bool {
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", msg+": "+err.Error(), "")
		return true
	}
	return false
}

// HandleErrorStatus is like HandleError but allows custom HTTP status.
func HandleErrorStatus(c *gin.Context, err error, msg string, status int) bool {
	if err != nil {
		RespondError(c, status, "ERROR", msg+": "+err.Error(), "")
		return true
	}
	return false
}

// RespondSuccess sends a standardized success response.
func RespondSuccess(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{Success: true, Data: data})
}

// RespondError sends a standardized error response.
func RespondError(c *gin.Context, status int, code, message, details string) {
	c.JSON(status, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message, Details: details},
	})
}

// RespondSuccessWithMeta sends a standardized success response with meta.
func RespondSuccessWithMeta(c *gin.Context, status int, data interface{}, meta *Meta) {
	c.JSON(status, APIResponse{Success: true, Data: data, Meta: meta})
}

// WithTx executes fn within a database transaction.
// Automatically rolls back on error or panic, commits on success.
// Usage:
//
//	err := WithTx(c.Request.Context(), func(tx *sql.Tx) error {
//	    // use tx for queries
//	    return nil
//	})
//	if HandleError(c, err, "Transaction failed") {
//	    return
//	}
func WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
