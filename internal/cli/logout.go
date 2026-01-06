package cli

import (
	"fmt"

	"github.com/dorinclisu/hass-cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	Long: `Remove the stored Home Assistant credentials.

This will delete the configuration file containing the server URL and access token.
You will need to run 'hass-cli login' again to use the CLI.

Note: This does not revoke the access token on the Home Assistant server.
To revoke the token, go to your Home Assistant profile and delete it there.`,
	RunE: runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	// Check if config exists
	_, err := config.LoadFrom(cfgPath)
	if err != nil {
		if err == config.ErrNotConfigured {
			printSuccess("Already logged out (no configuration found)")
			return nil
		}
		// If there's another error, still try to delete
		printInfo("Warning: could not read config: %v", err)
	}

	// Delete the configuration
	if err := config.DeleteFrom(cfgPath); err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	printSuccess("Successfully logged out")
	printSuccess("Configuration removed from %s", cfgPath)

	return nil
}
