package models

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMigrations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migrations_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test migration files
	upContent := "CREATE TABLE users (id INTEGER PRIMARY KEY);"
	downContent := "DROP TABLE users;"
	
	upFile := filepath.Join(tmpDir, "20230101120000_create_users.up.sql")
	downFile := filepath.Join(tmpDir, "20230101120000_create_users.down.sql")

	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		t.Fatalf("Failed to write up migration: %v", err)
	}

	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		t.Fatalf("Failed to write down migration: %v", err)
	}

	// Test loading migrations
	migrationSet, err := LoadMigrations(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrationSet.Migrations) != 1 {
		t.Fatalf("Expected 1 migration, got %d", len(migrationSet.Migrations))
	}

	migration := migrationSet.Migrations[0]
	if migration.Filename != "20230101120000_create_users" {
		t.Errorf("Expected filename 20230101120000_create_users, got %s", migration.Filename)
	}

	if migration.Up != upContent {
		t.Errorf("Expected up content %s, got %s", upContent, migration.Up)
	}

	if migration.Down != downContent {
		t.Errorf("Expected down content %s, got %s", downContent, migration.Down)
	}

	if migration.Checksum == "" {
		t.Error("Expected checksum to be generated")
	}
}

func TestLoadMigrationsEmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "migrations_empty_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	migrationSet, err := LoadMigrations(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load migrations from empty dir: %v", err)
	}

	if len(migrationSet.Migrations) != 0 {
		t.Errorf("Expected 0 migrations, got %d", len(migrationSet.Migrations))
	}
}

func TestLoadMigrationsNonexistentDir(t *testing.T) {
	migrationSet, err := LoadMigrations("/nonexistent/path")
	if err != nil {
		t.Fatalf("Failed to handle nonexistent dir: %v", err)
	}

	if len(migrationSet.Migrations) != 0 {
		t.Errorf("Expected 0 migrations, got %d", len(migrationSet.Migrations))
	}
}

func TestGetPending(t *testing.T) {
	migrationSet := &MigrationSet{
		Migrations: []Migration{
			{Filename: "20230101120000_migration1"},
			{Filename: "20230101130000_migration2"},
			{Filename: "20230101140000_migration3"},
		},
	}

	appliedMigrations := []string{
		"20230101120000_migration1",
	}

	pending := migrationSet.GetPending(appliedMigrations)

	if len(pending) != 2 {
		t.Fatalf("Expected 2 pending migrations, got %d", len(pending))
	}

	if pending[0].Filename != "20230101130000_migration2" {
		t.Errorf("Expected first pending migration 20230101130000_migration2, got %s", pending[0].Filename)
	}

	if pending[1].Filename != "20230101140000_migration3" {
		t.Errorf("Expected second pending migration 20230101140000_migration3, got %s", pending[1].Filename)
	}
}

func TestGetMigrationByFilename(t *testing.T) {
	migrationSet := &MigrationSet{
		Migrations: []Migration{
			{Filename: "20230101120000_migration1"},
			{Filename: "20230101130000_migration2"},
		},
	}

	migration, err := migrationSet.GetMigrationByFilename("20230101130000_migration2")
	if err != nil {
		t.Fatalf("Failed to get migration by filename: %v", err)
	}

	if migration.Filename != "20230101130000_migration2" {
		t.Errorf("Expected migration 20230101130000_migration2, got %s", migration.Filename)
	}

	_, err = migrationSet.GetMigrationByFilename("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent migration")
	}
}

func TestGenerateMigrationFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "generate_migrations_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	name := "create_users_table"
	err = GenerateMigrationFiles(tmpDir, name)
	if err != nil {
		t.Fatalf("Failed to generate migration files: %v", err)
	}

	// Check if files were created
	files, err := filepath.Glob(filepath.Join(tmpDir, "*_create_users_table.*"))
	if err != nil {
		t.Fatalf("Failed to list generated files: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	// Verify up file exists and has content
	upFiles, err := filepath.Glob(filepath.Join(tmpDir, "*_create_users_table.up.sql"))
	if err != nil {
		t.Fatalf("Failed to find up file: %v", err)
	}

	if len(upFiles) != 1 {
		t.Fatalf("Expected 1 up file, got %d", len(upFiles))
	}

	upContent, err := os.ReadFile(upFiles[0])
	if err != nil {
		t.Fatalf("Failed to read up file: %v", err)
	}

	if len(upContent) == 0 {
		t.Error("Up file should not be empty")
	}

	// Verify down file exists and has content
	downFiles, err := filepath.Glob(filepath.Join(tmpDir, "*_create_users_table.down.sql"))
	if err != nil {
		t.Fatalf("Failed to find down file: %v", err)
	}

	if len(downFiles) != 1 {
		t.Fatalf("Expected 1 down file, got %d", len(downFiles))
	}

	downContent, err := os.ReadFile(downFiles[0])
	if err != nil {
		t.Fatalf("Failed to read down file: %v", err)
	}

	if len(downContent) == 0 {
		t.Error("Down file should not be empty")
	}
}

func TestGenerateChecksum(t *testing.T) {
	content1 := "CREATE TABLE users (id INTEGER PRIMARY KEY);"
	content2 := "CREATE TABLE posts (id INTEGER PRIMARY KEY);"
	
	checksum1 := generateChecksum(content1)
	checksum2 := generateChecksum(content2)
	checksum3 := generateChecksum(content1) // Same as checksum1

	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	if checksum1 == checksum2 {
		t.Error("Different content should produce different checksums")
	}

	if checksum1 != checksum3 {
		t.Error("Same content should produce same checksums")
	}

	// Verify checksum format (MD5 hex)
	if len(checksum1) != 32 {
		t.Errorf("Expected checksum length 32, got %d", len(checksum1))
	}
}

func TestMigrationTimestampParsing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "timestamp_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create migrations with different timestamps
	timestamps := []string{
		"20230101120000",
		"20230102130000", 
		"20230103140000",
	}

	for _, ts := range timestamps {
		upFile := filepath.Join(tmpDir, ts+"_test.up.sql")
		downFile := filepath.Join(tmpDir, ts+"_test.down.sql")
		
		if err := os.WriteFile(upFile, []byte("-- up"), 0644); err != nil {
			t.Fatalf("Failed to write migration file: %v", err)
		}
		if err := os.WriteFile(downFile, []byte("-- down"), 0644); err != nil {
			t.Fatalf("Failed to write migration file: %v", err)
		}
	}

	migrationSet, err := LoadMigrations(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrationSet.Migrations) != 3 {
		t.Fatalf("Expected 3 migrations, got %d", len(migrationSet.Migrations))
	}

	// Verify migrations are sorted by timestamp
	for i := 1; i < len(migrationSet.Migrations); i++ {
		prev := migrationSet.Migrations[i-1]
		curr := migrationSet.Migrations[i]
		
		if !prev.Timestamp.Before(curr.Timestamp) {
			t.Errorf("Migrations not sorted properly: %v should be before %v", 
				prev.Timestamp, curr.Timestamp)
		}
	}

	// Verify specific timestamp parsing
	expected := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	if !migrationSet.Migrations[0].Timestamp.Equal(expected) {
		t.Errorf("Expected timestamp %v, got %v", expected, migrationSet.Migrations[0].Timestamp)
	}
}