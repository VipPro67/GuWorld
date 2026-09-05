package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

// HandleError is a helper to reduce error handling boilerplate.
// Returns true if error was handled (response sent), false if err is nil.
// Usage:
//
//	if HandleError(c, err, "Failed to do something") {
//	    return
//	}
func HandleError(c *gin.Context, err error, msg string) bool {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": msg + ": " + err.Error()})
		return true
	}
	return false
}

// HandleErrorStatus is like HandleError but allows custom HTTP status.
func HandleErrorStatus(c *gin.Context, err error, msg string, status int) bool {
	if err != nil {
		c.JSON(status, gin.H{"error": msg + ": " + err.Error()})
		return true
	}
	return false
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