package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"migr8/pkg/config"
	"migr8/pkg/database"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage Migr8 configuration files and settings.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Create a sample configuration file in the current directory.
This generates a .migr8.yaml file with default settings that you can customize.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := ".migr8.yaml"
		
		if _, err := os.Stat(configFile); err == nil {
			return fmt.Errorf("configuration file already exists: %s", configFile)
		}

		sampleConfig := config.Config{
			Database: config.DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Database: "your_database",
				Username: "your_username",
				Password: "your_password",
				SSLMode:  "disable",
			},
			Migration: config.MigrationConfig{
				Directory: "./migrations",
				Table:     "schema_migrations",
			},
			Backup: config.BackupConfig{
				Directory:     "./backups",
				Compression:   true,
				RetentionDays: 30,
			},
			Seed: config.SeedConfig{
				Directory: "./seeds",
			},
			Verbose: false,
		}

		yamlData, err := yaml.Marshal(sampleConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		header := `# Migr8 Configuration File
# 
# This file contains the configuration for Migr8 database operations.
# Customize the settings below to match your environment.
#
# Supported database drivers: postgres, mysql, sqlite3
# 
# You can also use environment variables:
# - MIGR8_DB_HOST, MIGR8_DB_PORT, MIGR8_DB_USER, etc.
#

`

		content := header + string(yamlData)

		if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Configuration file created: %s\n", configFile)
		fmt.Printf("Please edit the file to match your environment settings.\n")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings loaded from config file and environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Current Configuration:\n")
		fmt.Printf("=====================\n\n")

		fmt.Printf("Database:\n")
		fmt.Printf("  Driver:   %s\n", cfg.Database.Driver)
		fmt.Printf("  Host:     %s\n", cfg.Database.Host)
		fmt.Printf("  Port:     %d\n", cfg.Database.Port)
		fmt.Printf("  Database: %s\n", cfg.Database.Database)
		fmt.Printf("  Username: %s\n", cfg.Database.Username)
		fmt.Printf("  Password: %s\n", maskPassword(cfg.Database.Password))
		fmt.Printf("  SSL Mode: %s\n", cfg.Database.SSLMode)

		fmt.Printf("\nMigration:\n")
		fmt.Printf("  Directory: %s\n", cfg.Migration.Directory)
		fmt.Printf("  Table:     %s\n", cfg.Migration.Table)

		fmt.Printf("\nBackup:\n")
		fmt.Printf("  Directory:     %s\n", cfg.Backup.Directory)
		fmt.Printf("  Compression:   %t\n", cfg.Backup.Compression)
		fmt.Printf("  Retention:     %d days\n", cfg.Backup.RetentionDays)

		fmt.Printf("\nSeed:\n")
		fmt.Printf("  Directory: %s\n", cfg.Seed.Directory)

		fmt.Printf("\nOther:\n")
		fmt.Printf("  Verbose: %t\n", cfg.Verbose)

		return nil
	},
}

var configTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test database connection",
	Long:  `Test the database connection using current configuration settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Printf("Testing database connection...\n")
		fmt.Printf("Driver: %s\n", cfg.Database.Driver)
		fmt.Printf("Host: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
		fmt.Printf("Database: %s\n", cfg.Database.Database)

		db, err := database.NewConnection(cfg)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		defer db.Close()

		fmt.Printf("✓ Connection successful!\n")
		return nil
	},
}

func maskPassword(password string) string {
	if len(password) == 0 {
		return ""
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configTestCmd)
}