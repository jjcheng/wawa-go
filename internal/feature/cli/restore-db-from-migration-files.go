package feature_cli

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

func RestoreLocalDBFromMigrationFiles() {
	log.Println("========RESTORING LOCAL DB FROM MIGRATION FILES========")
	// Find and execute all .up.sql files
	migrationFiles, err := findUpSQLFiles("migration")
	if err != nil {
		log.Fatal("Failed to find migration files:", err)
	}
	if len(migrationFiles) == 0 {
		log.Println("⚠️  No .up.sql files found in migration folder")
		return
	}
	log.Printf("📋 Found %d migration files to execute:\n", len(migrationFiles))
	for _, file := range migrationFiles {
		log.Printf("  - %s\n", filepath.Base(file))
	}
	// Load environment variables
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}
	// Get database configuration from environment variables
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")
	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC", dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	log.Printf("Connecting to PostgreSQL database: %s@%s:%s/%s\n", dbUser, dbHost, dbPort, dbName)
	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("✅ Connected to PostgreSQL database successfully!")
	// Drop and recreate the database
	log.Println("🗑️  Dropping and recreating database")
	// First connect to postgres database to drop the target database
	postgresConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s TimeZone=UTC", dbHost, dbPort, dbUser, dbPassword, dbSSLMode)
	postgresDB, err := sql.Open("postgres", postgresConnStr)
	if err != nil {
		log.Fatal("Failed to connect to postgres database:", err)
	}
	defer postgresDB.Close()
	// Terminate all connections to the database first
	// Use parameterized query to prevent SQL injection
	terminateSQL := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`
	_, err = postgresDB.Exec(terminateSQL, dbName)
	if err != nil {
		log.Printf("⚠️  Warning: Failed to terminate connections: %v\n", err)
	}
	// Drop the database if it exists
	// Note: Database names cannot be parameterized in DROP/CREATE statements
	// So we need to validate the database name to prevent injection
	if !isValidDatabaseName(dbName) {
		log.Fatal("Invalid database name - contains potentially dangerous characters")
	}
	dropDBSQL := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	_, err = postgresDB.Exec(dropDBSQL)
	if err != nil {
		log.Fatal("Failed to drop database:", err)
	}
	// Create the database
	createDBSQL := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err = postgresDB.Exec(createDBSQL)
	if err != nil {
		log.Fatal("Failed to create database:", err)
	}
	log.Printf("✅ Database %s dropped and recreated successfully!\n", dbName)
	// Close postgres connection and reconnect to the target database
	postgresDB.Close()
	db.Close()
	// Reconnect to the newly created database
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to reconnect to database:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping recreated database:", err)
	}
	// Execute each migration file in order
	for i, file := range migrationFiles {
		log.Printf("📊 Executing migration %d/%d: %s\n", i+1, len(migrationFiles), filepath.Base(file))
		migrationSQL, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", file, err)
		}
		_, err = db.Exec(string(migrationSQL))
		if err != nil {
			log.Fatalf("Failed to execute migration %s: %v", file, err)
		}
		log.Printf("✅ Migration %s completed successfully!\n", filepath.Base(file))
	}
	log.Printf("🎉 All %d migrations completed! Your PostgreSQL database is ready.\n", len(migrationFiles))
	log.Println("📋 Database has been updated with all migration files.")
}

// findUpSQLFiles finds all .up.sql files in the specified directory and returns them sorted
func findUpSQLFiles(dir string) ([]string, error) {
	var upFiles []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		if strings.HasSuffix(fileName, ".up.sql") {
			fullPath := filepath.Join(dir, fileName)
			upFiles = append(upFiles, fullPath)
		}
	}
	// Sort files to ensure consistent execution order
	sort.Strings(upFiles)
	return upFiles, nil
}

// isValidDatabaseName validates that a database name contains only safe characters
// to prevent SQL injection in database creation/deletion statements
func isValidDatabaseName(name string) bool {
	// PostgreSQL database names can contain letters, digits, and underscores
	// They must start with a letter or underscore
	// Maximum length is 63 characters
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	// Check first character
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	// Check remaining characters
	for i := 1; i < len(name); i++ {
		char := name[i]
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return true
}
