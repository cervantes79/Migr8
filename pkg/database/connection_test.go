//go:build integration

package database

import (
	"os"
	"strconv"
	"testing"

	"migr8/pkg/config"
)

func getTestConfig(driver string) *config.Config {
	var cfg config.Config
	
	cfg.Database.Driver = driver
	cfg.Database.Host = getEnv("TEST_DB_HOST", "localhost")
	cfg.Database.Username = getEnv("TEST_DB_USER", "testuser")
	cfg.Database.Password = getEnv("TEST_DB_PASS", "testpass")
	cfg.Database.Database = getEnv("TEST_DB_NAME", "migr8_test")
	
	switch driver {
	case "postgres":
		cfg.Database.Port = getEnvInt("TEST_DB_PORT", 5432)
		cfg.Database.SSLMode = "disable"
	case "mysql":
		cfg.Database.Port = getEnvInt("TEST_DB_PORT", 3306)
	case "sqlite3":
		cfg.Database.Database = ":memory:"
		cfg.Database.Port = 0
	}
	
	cfg.Migration.Table = "test_migrations"
	
	return &cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func TestNewConnection(t *testing.T) {
	drivers := []string{"sqlite3"}
	
	// Only test postgres/mysql if running in CI environment
	if os.Getenv("CI") == "true" {
		drivers = append(drivers, "postgres", "mysql")
	}
	
	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			cfg := getTestConfig(driver)
			
			db, err := NewConnection(cfg)
			if err != nil {
				t.Fatalf("Failed to create connection for %s: %v", driver, err)
			}
			defer db.Close()
			
			if db.Driver != driver {
				t.Errorf("Expected driver %s, got %s", driver, db.Driver)
			}
			
			// Test ping
			if err := db.Ping(); err != nil {
				t.Errorf("Failed to ping %s database: %v", driver, err)
			}
		})
	}
}

func TestCreateMigrationsTable(t *testing.T) {
	drivers := []string{"sqlite3"}
	
	if os.Getenv("CI") == "true" {
		drivers = append(drivers, "postgres", "mysql")
	}
	
	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			cfg := getTestConfig(driver)
			
			db, err := NewConnection(cfg)
			if err != nil {
				t.Fatalf("Failed to create connection: %v", err)
			}
			defer db.Close()
			
			tableName := "test_migrations_" + driver
			err = db.CreateMigrationsTable(tableName)
			if err != nil {
				t.Fatalf("Failed to create migrations table: %v", err)
			}
			
			// Verify table exists by trying to query it
			_, err = db.Query("SELECT * FROM " + tableName + " LIMIT 1")
			if err != nil {
				t.Errorf("Migrations table not accessible: %v", err)
			}
			
			// Cleanup
			_, err = db.Exec("DROP TABLE " + tableName)
			if err != nil {
				t.Errorf("Failed to cleanup test table: %v", err)
			}
		})
	}
}

func TestMigrationOperations(t *testing.T) {
	drivers := []string{"sqlite3"}
	
	if os.Getenv("CI") == "true" {
		drivers = append(drivers, "postgres", "mysql")
	}
	
	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			cfg := getTestConfig(driver)
			
			db, err := NewConnection(cfg)
			if err != nil {
				t.Fatalf("Failed to create connection: %v", err)
			}
			defer db.Close()
			
			tableName := "test_migrations_ops_" + driver
			
			// Create migrations table
			err = db.CreateMigrationsTable(tableName)
			if err != nil {
				t.Fatalf("Failed to create migrations table: %v", err)
			}
			defer db.Exec("DROP TABLE " + tableName)
			
			// Test initial empty state
			migrations, err := db.GetAppliedMigrations(tableName)
			if err != nil {
				t.Fatalf("Failed to get applied migrations: %v", err)
			}
			
			if len(migrations) != 0 {
				t.Errorf("Expected 0 applied migrations, got %d", len(migrations))
			}
			
			// Record a migration
			filename := "20230101120000_test_migration"
			checksum := "abc123def456"
			
			err = db.RecordMigration(tableName, filename, checksum)
			if err != nil {
				t.Fatalf("Failed to record migration: %v", err)
			}
			
			// Verify migration was recorded
			migrations, err = db.GetAppliedMigrations(tableName)
			if err != nil {
				t.Fatalf("Failed to get applied migrations after recording: %v", err)
			}
			
			if len(migrations) != 1 {
				t.Errorf("Expected 1 applied migration, got %d", len(migrations))
			}
			
			if migrations[0] != filename {
				t.Errorf("Expected migration %s, got %s", filename, migrations[0])
			}
			
			// Record another migration
			filename2 := "20230102130000_another_migration"
			checksum2 := "def456ghi789"
			
			err = db.RecordMigration(tableName, filename2, checksum2)
			if err != nil {
				t.Fatalf("Failed to record second migration: %v", err)
			}
			
			// Verify both migrations
			migrations, err = db.GetAppliedMigrations(tableName)
			if err != nil {
				t.Fatalf("Failed to get applied migrations after second recording: %v", err)
			}
			
			if len(migrations) != 2 {
				t.Errorf("Expected 2 applied migrations, got %d", len(migrations))
			}
			
			// Remove a migration
			err = db.RemoveMigration(tableName, filename)
			if err != nil {
				t.Fatalf("Failed to remove migration: %v", err)
			}
			
			// Verify migration was removed
			migrations, err = db.GetAppliedMigrations(tableName)
			if err != nil {
				t.Fatalf("Failed to get applied migrations after removal: %v", err)
			}
			
			if len(migrations) != 1 {
				t.Errorf("Expected 1 applied migration after removal, got %d", len(migrations))
			}
			
			if migrations[0] != filename2 {
				t.Errorf("Expected remaining migration %s, got %s", filename2, migrations[0])
			}
		})
	}
}

func TestUnsupportedDriver(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "unsupported_driver",
		},
	}
	
	_, err := NewConnection(cfg)
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}

func TestCreateMigrationsTableUnsupportedDriver(t *testing.T) {
	// This test uses a mock DB with unsupported driver
	db := &DB{
		Driver: "unsupported",
	}
	
	err := db.CreateMigrationsTable("test_table")
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}