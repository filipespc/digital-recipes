package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/handlers"
	"digital-recipes/api-service/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Configure structured logging
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, defaulting to info")
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	database, err := db.NewConnection()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to database")
	}
	defer database.Close()

	// Run migrations
	migrationsDir := filepath.Join("db", "migrations")
	if err := database.RunMigrations(migrationsDir); err != nil {
		logrus.WithError(err).Fatal("Failed to run migrations")
	}

	// Initialize authentication configuration
	authConfig := middleware.NewAuthConfig()
	logrus.WithFields(logrus.Fields{
		"jwt_duration": authConfig.TokenDuration,
		"jwt_issuer":   authConfig.Issuer,
	}).Info("Authentication configured")

	r := gin.New()
	
	// Add core middleware (order matters!)
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.StructuredLoggingMiddleware())
	r.Use(middleware.SecurityLoggingMiddleware())
	r.Use(middleware.CreateGeneralRateLimit())
	r.Use(gin.Recovery())
	
	// Add CORS middleware with environment-based configuration
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000" // Development default
	}
	
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowed := false
		
		// Check if origin is in allowed list
		for _, allowedOrigin := range strings.Split(allowedOrigins, ",") {
			if strings.TrimSpace(allowedOrigin) == origin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}
		
		if !allowed && origin != "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Origin not allowed"})
			c.Abort()
			return
		}
		
		c.Next()
	})
	
	// Initialize storage service
	storageService, err := handlers.NewStorageService()
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize storage service, upload functionality will not be available")
		// Continue without storage service for now (graceful degradation)
	}

	// Initialize handlers
	recipeHandler := handlers.NewRecipeHandler(database, storageService)
	
	r.GET("/health", func(c *gin.Context) {
		// Check database health
		dbStatus := "healthy"
		if err := database.HealthCheck(); err != nil {
			dbStatus = "unhealthy"
			logrus.WithError(err).Error("Database health check failed")
		}

		// Check storage health if available
		storageStatus := "not_configured"
		if storageService != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := storageService.HealthCheck(ctx); err != nil {
				storageStatus = "unhealthy"
				logrus.WithError(err).Warn("Storage health check failed")
			} else {
				storageStatus = "healthy"
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "digital-recipes-api",
			"database": dbStatus,
			"storage":  storageStatus,
			"version":  "1.0.0",
		})
	})

	// Public API routes (no authentication required)
	public := r.Group("/api/v1")
	{
		public.GET("/recipes", recipeHandler.GetRecipes)
		public.GET("/recipes/:id", recipeHandler.GetRecipe)
	}

	// Protected API routes (authentication required)
	protected := r.Group("/api/v1")
	protected.Use(middleware.OptionalAuthMiddleware(authConfig)) // Optional for backwards compatibility
	{
		// Upload endpoints with additional rate limiting
		uploadGroup := protected.Group("/recipes")
		uploadGroup.Use(middleware.CreateUploadRateLimit())
		{
			uploadGroup.POST("/upload-request", recipeHandler.PostUploadRequest)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logrus.WithFields(logrus.Fields{
		"port":            port,
		"database_status": "connected",
		"storage_status":  func() string {
			if storageService != nil {
				return "configured"
			}
			return "not_configured"
		}(),
	}).Info("Starting Digital Recipes API service")

	if err := r.Run(":" + port); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}
}