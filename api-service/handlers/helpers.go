package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"digital-recipes/api-service/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// verifyUserAccess checks if the authenticated user matches the requested user_id
// This is a common authorization pattern used across multiple handlers
// Returns true if access is granted, false if denied (and sends appropriate error response)
func verifyUserAccess(c *gin.Context, requestedUserID int, logger *logrus.Entry) bool {
	// Get the authenticated user from context
	authenticatedUserID := middleware.GetUserID(c)

	// Check if user is authenticated
	if authenticatedUserID == 0 {
		logger.Warn("Authentication required but no user found in context")
		AuthenticationError(c, "authentication required")
		return false
	}

	// Check if authenticated user matches requested user
	if authenticatedUserID != requestedUserID {
		logger.WithFields(logrus.Fields{
			"authenticated_user": authenticatedUserID,
			"requested_user":     requestedUserID,
			"path":               c.Request.URL.Path,
			"method":             c.Request.Method,
		}).Warn("Unauthorized access attempt: user trying to access another user's resources")
		AuthorizationError(c, "access denied: cannot access other users' resources")
		return false
	}

	// Access granted
	logger.WithFields(logrus.Fields{
		"user_id": authenticatedUserID,
		"path":    c.Request.URL.Path,
	}).Debug("User access verified")

	return true
}

// parseUserIDFromQuery extracts and validates user_id from query parameters
// This is commonly needed in search and filter endpoints
// Returns the user_id and true if valid, 0 and false if invalid (sends error response)
func parseUserIDFromQuery(c *gin.Context) (int, bool) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		BadRequestError(c, "query parameter 'user_id' is required")
		return 0, false
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		BadRequestError(c, "invalid user_id parameter")
		return 0, false
	}

	if userID < 1 {
		BadRequestError(c, "user_id must be a positive integer")
		return 0, false
	}

	return userID, true
}

// validateSearchQuery validates and sanitizes search query parameters
// Returns the cleaned query and true if valid, empty string and false if invalid
func validateSearchQuery(c *gin.Context, required bool, minLength, maxLength int) (string, bool) {
	query := c.Query("q")

	// Check if required
	if required && query == "" {
		BadRequestError(c, "query parameter 'q' is required")
		return "", false
	}

	// If not required and empty, return as valid
	if query == "" {
		return "", true
	}

	// Trim and validate length
	query = strings.TrimSpace(query)
	if len(query) < minLength {
		BadRequestError(c, fmt.Sprintf("query must be at least %d characters long", minLength))
		return "", false
	}
	if len(query) > maxLength {
		BadRequestError(c, fmt.Sprintf("query must not exceed %d characters", maxLength))
		return "", false
	}

	// Check for potentially dangerous SQL patterns (defense in depth)
	lowerQuery := strings.ToLower(query)
	if strings.ContainsAny(query, ";'\"\\") ||
		strings.Contains(lowerQuery, "drop") ||
		strings.Contains(lowerQuery, "delete") ||
		strings.Contains(lowerQuery, "insert") ||
		strings.Contains(lowerQuery, "update") ||
		strings.Contains(lowerQuery, "select") {
		BadRequestError(c, "query contains invalid characters or SQL keywords")
		return "", false
	}

	return query, true
}