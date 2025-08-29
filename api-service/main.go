package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

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