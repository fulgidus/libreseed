package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version information - set at build time
	Version   = "0.1.0-alpha"
	GitCommit = "dev"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display the version, git commit, and build date of the seeder.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("LibreSeed Seeder\n")
		fmt.Printf("  Version:    %s\n", Version)
		fmt.Printf("  Git Commit: %s\n", GitCommit)
		fmt.Printf("  Build Date: %s\n", BuildDate)
		fmt.Printf("  Go Version: %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
