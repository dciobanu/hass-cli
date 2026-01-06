// Package cli implements the command-line interface for hass-cli.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	jsonOutput bool
	configPath string
	serverURL  string
	token      string
	timeout    int
	verbose    bool

	// Version is set from main
	version = "dev"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hass-cli",
	Short: "Command-line interface for Home Assistant",
	Long: `hass-cli is a command-line interface for interacting with Home Assistant.

It allows you to control devices, monitor states, call services, and more
directly from your terminal.

Get started by running:
  hass-cli login --url http://your-ha-instance:8123 --token YOUR_TOKEN`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string for the CLI.
func SetVersion(v string) {
	version = v
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: ~/.config/hass-cli/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&serverURL, "url", "", "Home Assistant server URL (overrides config)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Access token (overrides config)")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30, "Request timeout in seconds")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add version command
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("hass-cli version %s\n", version)
	},
}

// printError prints an error message to stderr.
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// printSuccess prints a success message.
func printSuccess(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// printInfo prints an info message (only in verbose mode).
func printInfo(format string, args ...interface{}) {
	if verbose {
		fmt.Printf(format+"\n", args...)
	}
}
