package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"migr8/pkg/config"
	"migr8/pkg/migration"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration operations",
	Long: `Manage database schema migrations with full up/down support.
Supports PostgreSQL, MySQL, and SQLite databases.`,
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply pending migrations",
	Long: `Apply all pending migrations to bring the database schema up to date.
Migrations are applied in chronological order based on timestamp.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		migrator, err := migration.NewMigrator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migrator: %w", err)
		}
		defer migrator.Close()

		return migrator.Up()
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down [steps]",
	Short: "Rollback migrations",
	Long: `Rollback the specified number of migrations.
If no steps specified, rolls back 1 migration.
Use 'all' to rollback all migrations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		steps := 1

		if len(args) > 0 {
			if args[0] == "all" {
				steps = 9999
			} else {
				var err error
				steps, err = strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("invalid steps argument: %s", args[0])
				}
				if steps <= 0 {
					return fmt.Errorf("steps must be a positive number")
				}
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		migrator, err := migration.NewMigrator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migrator: %w", err)
		}
		defer migrator.Close()

		return migrator.Down(steps)
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Long:  `Display the current status of all migrations, showing which have been applied.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		migrator, err := migration.NewMigrator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migrator: %w", err)
		}
		defer migrator.Close()

		return migrator.Status()
	},
}

var migrateCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new migration",
	Long: `Create a new migration with the specified name.
This generates both up and down migration files with timestamp prefixes.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		migrator, err := migration.NewMigrator(cfg)
		if err != nil {
			return fmt.Errorf("failed to create migrator: %w", err)
		}
		defer migrator.Close()

		if err := migrator.Create(args[0]); err != nil {
			return fmt.Errorf("failed to create migration: %w", err)
		}

		fmt.Printf("Created new migration: %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateCreateCmd)
}