package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "1.0.0"
	GitCommit = "dev"
	BuildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display detailed version information including build details.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Migr8 Database Migration Tool\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}