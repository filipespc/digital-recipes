package db

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// RunMigrations executes all pending migrations
func (d *Database) RunMigrations(migrationsDir string) error {
	// Create migrations table if it doesn't exist
	if err := d.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Load migration files
	migrations, err := loadMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	appliedVersions, err := d.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Execute pending migrations
	for _, migration := range migrations {
		if _, applied := appliedVersions[migration.Version]; applied {
			log.Printf("Migration %d_%s already applied, skipping", migration.Version, migration.Name)
			continue
		}

		log.Printf("Applying migration %d_%s", migration.Version, migration.Name)
		if err := d.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d_%s: %w", migration.Version, migration.Name, err)
		}
	}

	return nil
}

// createMigrationsTable creates the schema_migrations table
func (d *Database) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`
	
	_, err := d.DB.Exec(query)
	return err
}

// getAppliedMigrations returns a map of applied migration versions
func (d *Database) getAppliedMigrations() (map[int]bool, error) {
	query := "SELECT version FROM schema_migrations"
	rows, err := d.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// applyMigration executes a single migration
func (d *Database) applyMigration(migration Migration) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.Exec(migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", migration.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// loadMigrations loads migration files from the given directory
func loadMigrations(dir string) ([]Migration, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	migrationMap := make(map[int]*Migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		version, name, direction, err := parseMigrationFilename(filename)
		if err != nil {
			log.Printf("Skipping invalid migration file %s: %v", filename, err)
			continue
		}

		// Read file content
		content, err := ioutil.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Get or create migration struct
		if migrationMap[version] == nil {
			migrationMap[version] = &Migration{
				Version: version,
				Name:    name,
			}
		}

		// Set SQL content based on direction
		if direction == "up" {
			migrationMap[version].UpSQL = string(content)
		} else {
			migrationMap[version].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	for _, migration := range migrationMap {
		if migration.UpSQL == "" {
			return nil, fmt.Errorf("missing up migration for version %d", migration.Version)
		}
		migrations = append(migrations, *migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFilename parses migration filename like "001_initial_schema.up.sql"
func parseMigrationFilename(filename string) (version int, name string, direction string, err error) {
	// Remove .sql extension
	name = strings.TrimSuffix(filename, ".sql")
	
	// Split by dots to get direction
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return 0, "", "", fmt.Errorf("invalid migration filename format")
	}
	
	direction = parts[1]
	if direction != "up" && direction != "down" {
		return 0, "", "", fmt.Errorf("invalid migration direction: %s", direction)
	}
	
	// Split by underscore to get version and name
	nameParts := strings.SplitN(parts[0], "_", 2)
	if len(nameParts) < 2 {
		return 0, "", "", fmt.Errorf("invalid migration filename format")
	}
	
	version, err = strconv.Atoi(nameParts[0])
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid version number: %w", err)
	}
	
	name = nameParts[1]
	return version, name, direction, nil
}