package main

import (
	"flag"
	"log"
	"path/filepath"

	"digital-recipes/api-service/db"
)

func main() {
	var command = flag.String("command", "up", "Migration command: up, down, or status")
	flag.Parse()

	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	migrationsDir := filepath.Join("db", "migrations")

	switch *command {
	case "up":
		log.Println("Running migrations...")
		if err := database.RunMigrations(migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations completed successfully")
	case "status":
		log.Println("Migration status checking not implemented yet")
	case "down":
		log.Println("Rollback migrations not implemented yet")
	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}