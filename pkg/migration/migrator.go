package migration

import (
	"database/sql"
	"fmt"
	"strings"

	"migr8/internal/models"
	"migr8/pkg/config"
	"migr8/pkg/database"
)

type Migrator struct {
	db     *database.DB
	config *config.Config
}

func NewMigrator(cfg *config.Config) (*Migrator, error) {
	db, err := database.NewConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &Migrator{
		db:     db,
		config: cfg,
	}, nil
}

func (m *Migrator) Close() error {
	return m.db.Close()
}

func (m *Migrator) Up() error {
	if err := m.db.CreateMigrationsTable(m.config.Migration.Table); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrationSet, err := models.LoadMigrations(m.config.Migration.Directory)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	appliedMigrations, err := m.db.GetAppliedMigrations(m.config.Migration.Table)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pendingMigrations := migrationSet.GetPending(appliedMigrations)
	if len(pendingMigrations) == 0 {
		fmt.Println("No pending migrations found.")
		return nil
	}

	fmt.Printf("Applying %d pending migrations...\n", len(pendingMigrations))

	for _, migration := range pendingMigrations {
		if err := m.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Filename, err)
		}
		fmt.Printf("Applied migration: %s\n", migration.Filename)
	}

	fmt.Println("All migrations applied successfully!")
	return nil
}

func (m *Migrator) Down(steps int) error {
	appliedMigrations, err := m.db.GetAppliedMigrations(m.config.Migration.Table)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(appliedMigrations) == 0 {
		fmt.Println("No migrations to rollback.")
		return nil
	}

	migrationSet, err := models.LoadMigrations(m.config.Migration.Directory)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	if steps <= 0 || steps > len(appliedMigrations) {
		steps = len(appliedMigrations)
	}

	toRollback := appliedMigrations[len(appliedMigrations)-steps:]
	
	fmt.Printf("Rolling back %d migrations...\n", len(toRollback))

	for i := len(toRollback) - 1; i >= 0; i-- {
		migrationFilename := toRollback[i]
		migration, err := migrationSet.GetMigrationByFilename(migrationFilename)
		if err != nil {
			return fmt.Errorf("failed to find migration %s: %w", migrationFilename, err)
		}

		if migration.Down == "" {
			return fmt.Errorf("migration %s has no down migration", migrationFilename)
		}

		if err := m.rollbackMigration(*migration); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migrationFilename, err)
		}
		fmt.Printf("Rolled back migration: %s\n", migrationFilename)
	}

	fmt.Println("Rollback completed successfully!")
	return nil
}

func (m *Migrator) Status() error {
	migrationSet, err := models.LoadMigrations(m.config.Migration.Directory)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	appliedMigrations, err := m.db.GetAppliedMigrations(m.config.Migration.Table)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[string]bool)
	for _, applied := range appliedMigrations {
		appliedSet[applied] = true
	}

	fmt.Printf("Migration Status:\n")
	fmt.Printf("================\n\n")

	if len(migrationSet.Migrations) == 0 {
		fmt.Println("No migrations found.")
		return nil
	}

	for _, migration := range migrationSet.Migrations {
		status := "[ ]"
		if appliedSet[migration.Filename] {
			status = "[✓]"
		}
		fmt.Printf("%s %s\n", status, migration.Filename)
	}

	pendingCount := len(migrationSet.GetPending(appliedMigrations))
	fmt.Printf("\nTotal: %d migrations, %d applied, %d pending\n", 
		len(migrationSet.Migrations), len(appliedMigrations), pendingCount)

	return nil
}

func (m *Migrator) Create(name string) error {
	return models.GenerateMigrationFiles(m.config.Migration.Directory, name)
}

func (m *Migrator) applyMigration(migration models.Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	statements := strings.Split(migration.Up, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement '%s': %w", stmt, err)
		}
	}

	if err := m.recordMigrationInTx(tx, migration); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

func (m *Migrator) rollbackMigration(migration models.Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	statements := strings.Split(migration.Down, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement '%s': %w", stmt, err)
		}
	}

	if err := m.removeMigrationInTx(tx, migration.Filename); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit()
}

func (m *Migrator) recordMigrationInTx(tx *sql.Tx, migration models.Migration) error {
	query := fmt.Sprintf("INSERT INTO %s (filename, checksum) VALUES (?, ?)", m.config.Migration.Table)
	
	if m.db.Driver == "postgres" {
		query = fmt.Sprintf("INSERT INTO %s (filename, checksum) VALUES ($1, $2)", m.config.Migration.Table)
	}
	
	_, err := tx.Exec(query, migration.Filename, migration.Checksum)
	return err
}

func (m *Migrator) removeMigrationInTx(tx *sql.Tx, filename string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE filename = ?", m.config.Migration.Table)
	
	if m.db.Driver == "postgres" {
		query = fmt.Sprintf("DELETE FROM %s WHERE filename = $1", m.config.Migration.Table)
	}
	
	_, err := tx.Exec(query, filename)
	return err
}