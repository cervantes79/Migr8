package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"migr8/pkg/config"
	"migr8/pkg/seed"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Database seeding operations",
	Long: `Manage database seeding with support for YAML and CSV files.
Populate your database with test or initial data for development and testing.`,
}

var seedRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run all seed files",
	Long: `Execute all seed files found in the configured seed directory.
Processes both YAML and CSV files in alphabetical order.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		seeder, err := seed.NewSeeder(cfg)
		if err != nil {
			return fmt.Errorf("failed to create seeder: %w", err)
		}
		defer seeder.Close()

		fmt.Println("Running database seeds...")
		
		return seeder.Run()
	},
}

var seedGenerateCmd = &cobra.Command{
	Use:   "generate [table_name]",
	Short: "Generate a seed template",
	Long: `Generate a YAML seed template file for the specified table.
This creates a sample seed file that you can modify with your data.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		seeder, err := seed.NewSeeder(cfg)
		if err != nil {
			return fmt.Errorf("failed to create seeder: %w", err)
		}
		defer seeder.Close()

		tableName := args[0]
		fmt.Printf("Generating seed template for table: %s\n", tableName)
		
		if err := seeder.GenerateTemplate(tableName); err != nil {
			return fmt.Errorf("failed to generate seed template: %w", err)
		}

		fmt.Printf("Edit the generated file to add your seed data.\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
	seedCmd.AddCommand(seedRunCmd)
	seedCmd.AddCommand(seedGenerateCmd)
}