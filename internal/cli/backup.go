package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"migr8/pkg/backup"
	"migr8/pkg/config"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Database backup operations",
	Long: `Manage database backups with compression and retention policies.
Supports automated backups, listing, restoration, and cleanup operations.`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a database backup",
	Long: `Create a new backup of the configured database.
Supports optional compression and automatic timestamping.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		backupManager, err := backup.NewBackupManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create backup manager: %w", err)
		}
		defer backupManager.Close()

		fmt.Println("Creating database backup...")
		
		backupInfo, err := backupManager.Create()
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		compressionStatus := "uncompressed"
		if backupInfo.Compressed {
			compressionStatus = "compressed"
		}

		fmt.Printf("Backup created successfully!\n")
		fmt.Printf("File: %s\n", backupInfo.Filename)
		fmt.Printf("Path: %s\n", backupInfo.Path)
		fmt.Printf("Size: %.2f MB (%s)\n", float64(backupInfo.Size)/(1024*1024), compressionStatus)
		fmt.Printf("Database: %s\n", backupInfo.DatabaseName)

		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Long:  `Display all available database backups with their details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		backupManager, err := backup.NewBackupManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create backup manager: %w", err)
		}
		defer backupManager.Close()

		backups, err := backupManager.List()
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		if len(backups) == 0 {
			fmt.Println("No backups found.")
			return nil
		}

		fmt.Printf("Available Backups:\n")
		fmt.Printf("==================\n\n")
		fmt.Printf("%-30s %-15s %-10s %-12s %s\n", "Filename", "Database", "Size", "Compression", "Created")
		fmt.Printf("%s\n", strings.Repeat("-", 80))

		for _, backup := range backups {
			compressionStatus := "No"
			if backup.Compressed {
				compressionStatus = "Yes"
			}

			sizeStr := fmt.Sprintf("%.1f MB", float64(backup.Size)/(1024*1024))
			
			fmt.Printf("%-30s %-15s %-10s %-12s %s\n",
				backup.Filename,
				backup.DatabaseName,
				sizeStr,
				compressionStatus,
				backup.CreatedAt.Format("2006-01-02 15:04"),
			)
		}

		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [backup_file]",
	Short: "Restore from a backup",
	Long: `Restore the database from the specified backup file.
The backup file should be located in the configured backup directory.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		backupManager, err := backup.NewBackupManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create backup manager: %w", err)
		}
		defer backupManager.Close()

		backupFile := args[0]
		fmt.Printf("Restoring database from backup: %s\n", backupFile)
		
		if err := backupManager.Restore(backupFile); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}

		fmt.Println("Database restored successfully!")
		return nil
	},
}

var backupCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean old backups",
	Long: `Remove old backups based on the configured retention policy.
Backups older than the retention period will be permanently deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		backupManager, err := backup.NewBackupManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to create backup manager: %w", err)
		}
		defer backupManager.Close()

		fmt.Printf("Cleaning backups older than %d days...\n", cfg.Backup.RetentionDays)
		
		return backupManager.CleanOld()
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupCleanCmd)
}