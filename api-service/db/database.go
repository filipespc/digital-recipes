package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Database represents our database connection
type Database struct {
	DB *sql.DB
}

// NewConnection creates a new database connection
func NewConnection() (*Database, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool for optimal performance and resource management
	db.SetMaxOpenConns(25)                 // Maximum number of open connections to the database
	db.SetMaxIdleConns(10)                 // Maximum number of connections in the idle connection pool
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum amount of time a connection may be reused

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully")
	return &Database{DB: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d.DB != nil {
		return d.DB.Close()
	}
	return nil
}

// HealthCheck verifies database connectivity
func (d *Database) HealthCheck() error {
	return d.DB.Ping()
}