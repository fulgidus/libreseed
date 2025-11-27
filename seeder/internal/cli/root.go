package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/fulgidus/libreseed/seeder/internal/config"
	"github.com/fulgidus/libreseed/seeder/internal/logging"
)

var (
	cfgFile string
	logger  *zap.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "seeder",
	Short: "LibreSeed Seeder - Decentralized package distribution",
	Long: `LibreSeed Seeder is a BitTorrent-based content seeding service
that enables decentralized distribution of packages and content.

It supports:
  - DHT for peer discovery
  - Content manifest management
  - Multi-torrent seeding
  - Bandwidth and storage management`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger first
		var err error
		logger, err = logging.NewLogger(viper.GetString("log.level"), viper.GetString("log.format"))
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logger != nil {
			_ = logger.Sync()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./seeder.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "json", "log format (json, console)")

	// Bind flags to viper
	_ = viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("log.format", rootCmd.PersistentFlags().Lookup("log-format"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigName("seeder")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("SEEDER")
	viper.AutomaticEnv()

	// Load configuration using the config loader
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		// Continue with defaults
	}

	// If config was loaded successfully, merge it into viper
	if cfg != nil {
		// Viper already has the config loaded, just log success
		if viper.ConfigFileUsed() != "" {
			fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
		}
	}
}
