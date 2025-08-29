package models

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Migration struct {
	Filename  string
	Filepath  string
	Up        string
	Down      string
	Checksum  string
	Timestamp time.Time
}

type MigrationSet struct {
	Migrations []Migration
}

func LoadMigrations(directory string) (*MigrationSet, error) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return &MigrationSet{}, nil
	}

	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	var migrations []Migration
	migrationRegex := regexp.MustCompile(`^(\d{14})_(.+)\.(up|down)\.sql$`)

	migrationMap := make(map[string]*Migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		matches := migrationRegex.FindStringSubmatch(file.Name())
		if len(matches) != 4 {
			continue
		}

		timestamp := matches[1]
		name := matches[2]
		direction := matches[3]
		
		baseFilename := fmt.Sprintf("%s_%s", timestamp, name)
		
		content, err := os.ReadFile(filepath.Join(directory, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		if _, exists := migrationMap[baseFilename]; !exists {
			parsedTime, _ := time.Parse("20060102150405", timestamp)
			migrationMap[baseFilename] = &Migration{
				Filename:  baseFilename,
				Filepath:  filepath.Join(directory, file.Name()),
				Timestamp: parsedTime,
			}
		}

		migration := migrationMap[baseFilename]
		contentStr := string(content)
		
		if direction == "up" {
			migration.Up = contentStr
		} else {
			migration.Down = contentStr
		}
	}

	for _, migration := range migrationMap {
		if migration.Up == "" {
			return nil, fmt.Errorf("missing up migration for %s", migration.Filename)
		}
		
		migration.Checksum = generateChecksum(migration.Up)
		migrations = append(migrations, *migration)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Timestamp.Before(migrations[j].Timestamp)
	})

	return &MigrationSet{Migrations: migrations}, nil
}

func (ms *MigrationSet) GetPending(appliedMigrations []string) []Migration {
	appliedSet := make(map[string]bool)
	for _, applied := range appliedMigrations {
		appliedSet[applied] = true
	}

	var pending []Migration
	for _, migration := range ms.Migrations {
		if !appliedSet[migration.Filename] {
			pending = append(pending, migration)
		}
	}

	return pending
}

func (ms *MigrationSet) GetMigrationByFilename(filename string) (*Migration, error) {
	for _, migration := range ms.Migrations {
		if migration.Filename == filename {
			return &migration, nil
		}
	}
	return nil, fmt.Errorf("migration not found: %s", filename)
}

func GenerateMigrationFiles(directory, name string) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	cleanName := strings.ReplaceAll(strings.ToLower(name), " ", "_")
	
	baseFilename := fmt.Sprintf("%s_%s", timestamp, cleanName)
	
	upFile := filepath.Join(directory, baseFilename+".up.sql")
	downFile := filepath.Join(directory, baseFilename+".down.sql")

	upTemplate := fmt.Sprintf(`-- Migration: %s
-- Created: %s
-- Description: Add your up migration here

-- Example:
-- CREATE TABLE users (
--     id SERIAL PRIMARY KEY,
--     email VARCHAR(255) UNIQUE NOT NULL,
--     created_at TIMESTAMP DEFAULT NOW()
-- );
`, name, time.Now().Format("2006-01-02 15:04:05"))

	downTemplate := fmt.Sprintf(`-- Migration: %s (Down)
-- Created: %s
-- Description: Add your down migration here

-- Example:
-- DROP TABLE IF EXISTS users;
`, name, time.Now().Format("2006-01-02 15:04:05"))

	if err := os.WriteFile(upFile, []byte(upTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create up migration file: %w", err)
	}

	if err := os.WriteFile(downFile, []byte(downTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create down migration file: %w", err)
	}

	return nil
}

func generateChecksum(content string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(content)))
}