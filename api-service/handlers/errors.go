package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization  ErrorType = "authorization"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeExternal       ErrorType = "external"
)

// AppError represents a structured application error
type AppError struct {
	Type    ErrorType `json:"type"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Field   string    `json:"field,omitempty"`
}

// Error implements the error interface
func (e AppError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(code, message, field string) AppError {
	return AppError{
		Type:    ErrorTypeValidation,
		Code:    code,
		Message: message,
		Field:   field,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(code, message string) AppError {
	return AppError{
		Type:    ErrorTypeAuthentication,
		Code:    code,
		Message: message,
	}
}

// NewInternalError creates a new internal error with safe message
func NewInternalError(code, publicMessage string) AppError {
	return AppError{
		Type:    ErrorTypeInternal,
		Code:    code,
		Message: publicMessage,
	}
}

// SafeErrorResponse sends an error response with safe error messages
func SafeErrorResponse(c *gin.Context, err error, statusCode int) {
	requestID := c.GetHeader("X-Request-ID")
	userID := getUserIDSafe(c)

	// Log the actual error with full details
	logFields := logrus.Fields{
		"request_id": requestID,
		"user_id":    userID,
		"ip":         c.ClientIP(),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"error":      err.Error(),
		"status":     statusCode,
	}

	// Check if it's an AppError
	if appErr, ok := err.(AppError); ok {
		logFields["error_type"] = appErr.Type
		logFields["error_code"] = appErr.Code

		// Log at appropriate level based on error type
		switch appErr.Type {
		case ErrorTypeValidation, ErrorTypeAuthentication, ErrorTypeAuthorization, ErrorTypeNotFound:
			logrus.WithFields(logFields).Warn("Client error occurred")
		default:
			logrus.WithFields(logFields).Error("Application error occurred")
		}

		c.JSON(statusCode, gin.H{
			"error":      appErr.Message,
			"type":       appErr.Type,
			"code":       appErr.Code,
			"request_id": requestID,
		})
		return
	}

	// For unknown errors, log full details but return safe message
	logrus.WithFields(logFields).Error("Unhandled error occurred")

	// Return generic error message to prevent information leakage
	c.JSON(statusCode, gin.H{
		"error":      getSafeErrorMessage(statusCode),
		"request_id": requestID,
	})
}

// Enhanced response functions with better error handling
func ValidationError(c *gin.Context, message string, field ...string) {
	fieldName := ""
	if len(field) > 0 {
		fieldName = field[0]
	}
	err := NewValidationError("INVALID_INPUT", message, fieldName)
	SafeErrorResponse(c, err, http.StatusBadRequest)
}

func AuthenticationError(c *gin.Context, message string) {
	err := NewAuthenticationError("AUTH_REQUIRED", message)
	SafeErrorResponse(c, err, http.StatusUnauthorized)
}

func AuthorizationError(c *gin.Context, message string) {
	err := AppError{
		Type:    ErrorTypeAuthorization,
		Code:    "FORBIDDEN",
		Message: message,
	}
	SafeErrorResponse(c, err, http.StatusForbidden)
}

func NotFoundError(c *gin.Context, message string) {
	err := AppError{
		Type:    ErrorTypeNotFound,
		Code:    "NOT_FOUND",
		Message: message,
	}
	SafeErrorResponse(c, err, http.StatusNotFound)
}

func ConflictError(c *gin.Context, message string) {
	err := AppError{
		Type:    ErrorTypeConflict,
		Code:    "CONFLICT",
		Message: message,
	}
	SafeErrorResponse(c, err, http.StatusConflict)
}

func InternalServerError(c *gin.Context, publicMessage string) {
	err := NewInternalError("INTERNAL_ERROR", publicMessage)
	SafeErrorResponse(c, err, http.StatusInternalServerError)
}

// DatabaseError specifically handles database errors with proper classification
func DatabaseError(c *gin.Context, dbErr error, operation string) {
	requestID := c.GetHeader("X-Request-ID")
	userID := getUserIDSafe(c)

	// Log detailed database error
	logrus.WithFields(logrus.Fields{
		"request_id": requestID,
		"user_id":    userID,
		"ip":         c.ClientIP(),
		"operation":  operation,
		"db_error":   dbErr.Error(),
	}).Error("Database error occurred")

	// Classify database errors
	errorMsg := dbErr.Error()
	switch {
	case strings.Contains(errorMsg, "duplicate key"):
		ConflictError(c, "Resource already exists")
	case strings.Contains(errorMsg, "foreign key"):
		ValidationError(c, "Invalid reference to related resource")
	case strings.Contains(errorMsg, "not null constraint"):
		ValidationError(c, "Required field is missing")
	case strings.Contains(errorMsg, "connection"):
		InternalServerError(c, "Service temporarily unavailable")
	default:
		InternalServerError(c, "Database operation failed")
	}
}

// StorageError handles storage service errors
func StorageError(c *gin.Context, storageErr error, operation string) {
	requestID := c.GetHeader("X-Request-ID")
	userID := getUserIDSafe(c)

	logrus.WithFields(logrus.Fields{
		"request_id":    requestID,
		"user_id":       userID,
		"ip":            c.ClientIP(),
		"operation":     operation,
		"storage_error": storageErr.Error(),
	}).Error("Storage service error occurred")

	// Return user-friendly message without exposing internal details
	err := AppError{
		Type:    ErrorTypeExternal,
		Code:    "STORAGE_ERROR",
		Message: "File storage service is temporarily unavailable",
	}
	SafeErrorResponse(c, err, http.StatusInternalServerError)
}

// getUserIDSafe safely extracts user ID for logging
func getUserIDSafe(c *gin.Context) int {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int); ok {
			return id
		}
	}
	return 0
}

// getSafeErrorMessage returns generic error messages to prevent information disclosure
func getSafeErrorMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Invalid request"
	case http.StatusUnauthorized:
		return "Authentication required"
	case http.StatusForbidden:
		return "Access forbidden"
	case http.StatusNotFound:
		return "Resource not found"
	case http.StatusConflict:
		return "Resource conflict"
	case http.StatusTooManyRequests:
		return "Too many requests"
	case http.StatusInternalServerError:
		return "Internal server error"
	default:
		return "An error occurred"
	}
}