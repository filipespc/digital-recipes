package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Rate   limiter.Rate
	Store  limiter.Store
	KeyGen func(c *gin.Context) string
}

// NewMemoryRateLimit creates an in-memory rate limiter
func NewMemoryRateLimit(rate string) (*RateLimitConfig, error) {
	parsedRate, err := limiter.NewRateFromFormatted(rate)
	if err != nil {
		return nil, fmt.Errorf("invalid rate format: %w", err)
	}

	store := memory.NewStore()

	return &RateLimitConfig{
		Rate:  parsedRate,
		Store: store,
		KeyGen: func(c *gin.Context) string {
			// Rate limit by IP address
			return c.ClientIP()
		},
	}, nil
}

// NewUserRateLimit creates a rate limiter that limits by user ID
func NewUserRateLimit(rate string) (*RateLimitConfig, error) {
	parsedRate, err := limiter.NewRateFromFormatted(rate)
	if err != nil {
		return nil, fmt.Errorf("invalid rate format: %w", err)
	}

	store := memory.NewStore()

	return &RateLimitConfig{
		Rate:  parsedRate,
		Store: store,
		KeyGen: func(c *gin.Context) string {
			// Rate limit by user ID if authenticated, otherwise by IP
			if userID := GetUserID(c); userID != 0 {
				return fmt.Sprintf("user:%d", userID)
			}
			return fmt.Sprintf("ip:%s", c.ClientIP())
		},
	}, nil
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config *RateLimitConfig) gin.HandlerFunc {
	instance := limiter.New(config.Store, config.Rate)

	return func(c *gin.Context) {
		key := config.KeyGen(c)
		
		context, err := instance.Get(c, key)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error":      err.Error(),
				"key":        key,
				"ip":         c.ClientIP(),
				"request_id": c.GetHeader("X-Request-ID"),
			}).Error("Rate limiter error")
			
			// Continue on error to avoid blocking legitimate requests
			c.Next()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))

		if context.Reached {
			logrus.WithFields(logrus.Fields{
				"key":        key,
				"limit":      context.Limit,
				"ip":         c.ClientIP(),
				"path":       c.Request.URL.Path,
				"method":     c.Request.Method,
				"user_id":    GetUserID(c),
				"request_id": c.GetHeader("X-Request-ID"),
			}).Warn("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Limit: %d requests per %s", context.Limit, config.Rate.Period),
				"retry_after": context.Reset,
			})
			c.Abort()
			return
		}

		// Log rate limit usage for monitoring
		if context.Remaining < context.Limit/10 { // Warn when < 10% remaining
			logrus.WithFields(logrus.Fields{
				"key":       key,
				"remaining": context.Remaining,
				"limit":     context.Limit,
				"user_id":   GetUserID(c),
				"ip":        c.ClientIP(),
			}).Debug("Rate limit approaching")
		}

		c.Next()
	}
}

// CreateUploadRateLimit creates a specific rate limiter for upload endpoints
func CreateUploadRateLimit() gin.HandlerFunc {
	// Stricter limits for upload endpoints: 5 requests per minute per user
	config, err := NewUserRateLimit("5-M") // 5 requests per minute
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create upload rate limiter")
	}

	return RateLimitMiddleware(config)
}

// CreateGeneralRateLimit creates a general rate limiter for all endpoints
func CreateGeneralRateLimit() gin.HandlerFunc {
	// General rate limit: 100 requests per minute per IP
	config, err := NewMemoryRateLimit("100-M") // 100 requests per minute
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create general rate limiter")
	}

	return RateLimitMiddleware(config)
}

// CreateAuthRateLimit creates a rate limiter for authentication endpoints
func CreateAuthRateLimit() gin.HandlerFunc {
	// Auth endpoints: 10 requests per minute per IP (prevent brute force)
	config, err := NewMemoryRateLimit("10-M") // 10 requests per minute
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create auth rate limiter")
	}

	return RateLimitMiddleware(config)
}