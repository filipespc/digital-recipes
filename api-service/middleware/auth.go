package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Claims defines the JWT claims structure
type Claims struct {
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     string
	TokenDuration time.Duration
	Issuer        string
}

// NewAuthConfig creates a new auth configuration
func NewAuthConfig() *AuthConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Development fallback - in production this should be required
		secret = "dev-secret-key-change-in-production"
		logrus.Warn("JWT_SECRET not set, using development fallback")
	}

	durationStr := os.Getenv("JWT_DURATION")
	duration := 24 * time.Hour // Default 24 hours
	if durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}

	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "digital-recipes-api"
	}

	return &AuthConfig{
		JWTSecret:     secret,
		TokenDuration: duration,
		Issuer:        issuer,
	}
}

// AuthMiddleware creates JWT authentication middleware
func AuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logrus.WithFields(logrus.Fields{
				"ip":         c.ClientIP(),
				"path":       c.Request.URL.Path,
				"method":     c.Request.Method,
				"request_id": c.GetHeader("X-Request-ID"),
			}).Warn("Missing authorization header")
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logrus.WithFields(logrus.Fields{
				"ip":         c.ClientIP(),
				"request_id": c.GetHeader("X-Request-ID"),
			}).Warn("Invalid authorization header format")
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.JWTSecret), nil
		})

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ip":         c.ClientIP(),
				"error":      err.Error(),
				"request_id": c.GetHeader("X-Request-ID"),
			}).Warn("JWT parsing failed")
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			logrus.WithFields(logrus.Fields{
				"ip":         c.ClientIP(),
				"request_id": c.GetHeader("X-Request-ID"),
			}).Warn("Invalid JWT claims")
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
			})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)

		logrus.WithFields(logrus.Fields{
			"user_id":    claims.UserID,
			"user_email": claims.Email,
			"ip":         c.ClientIP(),
			"request_id": c.GetHeader("X-Request-ID"),
		}).Debug("Authentication successful")

		c.Next()
	}
}

// OptionalAuthMiddleware provides authentication but doesn't require it (for backwards compatibility)
func OptionalAuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to authenticate if header is present
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header - use default user for MVP compatibility
			c.Set("user_id", 1)
			c.Set("user_email", "mvp-user@example.com")
			c.Set("user_name", "MVP User")
			c.Next()
			return
		}

		// Run auth middleware logic
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			// Invalid format - use default user
			c.Set("user_id", 1)
			c.Set("user_email", "mvp-user@example.com")
			c.Set("user_name", "MVP User")
			c.Next()
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.JWTSecret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(*Claims); ok {
				c.Set("user_id", claims.UserID)
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Name)
			}
		} else {
			// Invalid token - use default user for MVP
			c.Set("user_id", 1)
			c.Set("user_email", "mvp-user@example.com")
			c.Set("user_name", "MVP User")
		}

		c.Next()
	}
}

// GenerateToken creates a new JWT token for a user
func GenerateToken(config *AuthConfig, userID int, email, name string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    config.Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.JWTSecret))
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) int {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int); ok {
			return id
		}
	}
	return 0 // Should not happen with proper middleware
}

// GetUserEmail extracts user email from gin context
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get("user_email"); exists {
		if emailStr, ok := email.(string); ok {
			return emailStr
		}
	}
	return ""
}

// GetUserName extracts user name from gin context
func GetUserName(c *gin.Context) string {
	if name, exists := c.Get("user_name"); exists {
		if nameStr, ok := name.(string); ok {
			return nameStr
		}
	}
	return ""
}