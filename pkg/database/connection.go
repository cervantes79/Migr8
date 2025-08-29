package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"migr8/pkg/config"
)

type DB struct {
	*sql.DB
	Driver string
}

func NewConnection(cfg *config.Config) (*DB, error) {
	dsn := cfg.GetDSN()
	if dsn == "" {
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	sqlDB, err := sql.Open(cfg.Database.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		DB:     sqlDB,
		Driver: cfg.Database.Driver,
	}, nil
}

func (db *DB) CreateMigrationsTable(tableName string) error {
	var query string
	
	switch db.Driver {
	case "postgres":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id SERIAL PRIMARY KEY,
				filename VARCHAR(255) NOT NULL UNIQUE,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				checksum VARCHAR(32) NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_%s_filename ON %s(filename);
		`, tableName, tableName, tableName)
	case "mysql":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INT AUTO_INCREMENT PRIMARY KEY,
				filename VARCHAR(255) NOT NULL UNIQUE,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				checksum VARCHAR(32) NOT NULL,
				INDEX idx_%s_filename (filename)
			) ENGINE=InnoDB;
		`, tableName, tableName)
	case "sqlite3":
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				filename TEXT NOT NULL UNIQUE,
				applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				checksum TEXT NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_%s_filename ON %s(filename);
		`, tableName, tableName, tableName)
	default:
		return fmt.Errorf("unsupported database driver: %s", db.Driver)
	}

	_, err := db.Exec(query)
	return err
}

func (db *DB) GetAppliedMigrations(tableName string) ([]string, error) {
	query := fmt.Sprintf("SELECT filename FROM %s ORDER BY id", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		migrations = append(migrations, filename)
	}

	return migrations, rows.Err()
}

func (db *DB) RecordMigration(tableName, filename, checksum string) error {
	query := fmt.Sprintf("INSERT INTO %s (filename, checksum) VALUES (?, ?)", tableName)
	
	if db.Driver == "postgres" {
		query = fmt.Sprintf("INSERT INTO %s (filename, checksum) VALUES ($1, $2)", tableName)
	}
	
	_, err := db.Exec(query, filename, checksum)
	return err
}

func (db *DB) RemoveMigration(tableName, filename string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE filename = ?", tableName)
	
	if db.Driver == "postgres" {
		query = fmt.Sprintf("DELETE FROM %s WHERE filename = $1", tableName)
	}
	
	_, err := db.Exec(query, filename)
	return err
}