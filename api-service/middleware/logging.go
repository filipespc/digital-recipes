package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists (from client or load balancer)
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate new UUID if not provided
			requestID = uuid.New().String()
		}

		// Set request ID in headers and context
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		c.Next()
	}
}

// StructuredLoggingMiddleware provides structured logging for all requests
func StructuredLoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Use structured logging instead of default gin format
			fields := logrus.Fields{
				"timestamp":   param.TimeStamp.Format(time.RFC3339),
				"status":      param.StatusCode,
				"latency":     param.Latency,
				"ip":          param.ClientIP,
				"method":      param.Method,
				"path":        param.Path,
				"user_agent":  param.Request.UserAgent(),
				"request_id":  param.Request.Header.Get("X-Request-ID"),
			}

			// Add user information if available
			if param.Request.Context().Value("user_id") != nil {
				fields["user_id"] = param.Request.Context().Value("user_id")
			}

			// Add error information if present
			if param.ErrorMessage != "" {
				fields["error"] = param.ErrorMessage
				logrus.WithFields(fields).Error("Request completed with error")
			} else {
				// Log different levels based on status code
				switch {
				case param.StatusCode >= 500:
					logrus.WithFields(fields).Error("Request completed")
				case param.StatusCode >= 400:
					logrus.WithFields(fields).Warn("Request completed")
				default:
					logrus.WithFields(fields).Info("Request completed")
				}
			}

			// Return empty string since we're using logrus for output
			return ""
		},
		Output:    logrus.StandardLogger().Out,
		SkipPaths: []string{"/health"}, // Skip health check logs to reduce noise
	})
}

// SecurityLoggingMiddleware logs security-relevant events
func SecurityLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Log request start for high-value endpoints
		if isHighValueEndpoint(c.Request.URL.Path) {
			logrus.WithFields(logrus.Fields{
				"request_id":    c.GetHeader("X-Request-ID"),
				"ip":           c.ClientIP(),
				"method":       c.Request.Method,
				"path":         c.Request.URL.Path,
				"user_agent":   c.Request.UserAgent(),
				"content_type": c.Request.Header.Get("Content-Type"),
				"user_id":      GetUserID(c),
			}).Info("High-value endpoint accessed")
		}

		c.Next()

		// Log security events based on response
		duration := time.Since(start)
		status := c.Writer.Status()

		securityFields := logrus.Fields{
			"request_id": c.GetHeader("X-Request-ID"),
			"ip":         c.ClientIP(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     status,
			"duration":   duration,
			"user_id":    GetUserID(c),
		}

		switch {
		case status == 401:
			logrus.WithFields(securityFields).Warn("Unauthorized access attempt")
		case status == 403:
			logrus.WithFields(securityFields).Warn("Forbidden access attempt")
		case status == 429:
			logrus.WithFields(securityFields).Warn("Rate limit exceeded")
		case status >= 500:
			logrus.WithFields(securityFields).Error("Server error occurred")
		case duration > 10*time.Second:
			logrus.WithFields(securityFields).Warn("Slow request detected")
		}
	}
}

// isHighValueEndpoint determines if an endpoint requires extra security logging
func isHighValueEndpoint(path string) bool {
	highValuePaths := []string{
		"/api/v1/recipes/upload-request",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/users",
	}

	for _, hvPath := range highValuePaths {
		if path == hvPath {
			return true
		}
	}
	return false
}

// GetRequestID extracts request ID from gin context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return c.GetHeader("X-Request-ID")
}

// LogWithContext creates a logrus entry with standard context fields
func LogWithContext(c *gin.Context) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"request_id": GetRequestID(c),
		"user_id":    GetUserID(c),
		"ip":         c.ClientIP(),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
	})
}