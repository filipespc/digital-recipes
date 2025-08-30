package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital-recipes/api-service/db"
	"digital-recipes/api-service/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	migrationsDir := filepath.Join("db", "migrations")
	if err := database.RunMigrations(migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	r := gin.Default()
	
	// Add request logging middleware
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		
		c.Next()
		
		// Log request details
		end := time.Now()
		latency := end.Sub(start)
		status := c.Writer.Status()
		
		log.Printf("[%s] %s %s - Status: %d - Duration: %v - IP: %s",
			method, path, c.Request.URL.RawQuery, status, latency, c.ClientIP())
	})
	
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
	
	// Initialize handlers
	recipeHandler := handlers.NewRecipeHandler(database)
	
	r.GET("/health", func(c *gin.Context) {
		// Check database health
		dbStatus := "healthy"
		if err := database.HealthCheck(); err != nil {
			dbStatus = "unhealthy"
			log.Printf("Database health check failed: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"service":  "digital-recipes-api",
			"database": dbStatus,
		})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.GET("/recipes", recipeHandler.GetRecipes)
		v1.GET("/recipes/:id", recipeHandler.GetRecipe)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Digital Recipes API service on :%s", port)
	log.Fatal(r.Run(":" + port))
}