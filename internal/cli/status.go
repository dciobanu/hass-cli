package cli

import (
	"fmt"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check Home Assistant API connectivity",
	Long: `Check the connection to Home Assistant and display system information.

Shows the Home Assistant version, location name, time zone, and other configuration details.

Examples:
  hass-cli status              # Check connectivity and show system info
  hass-cli status --json       # Output as JSON`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Checking connection to %s...", cfg.Server.URL)

	config, err := client.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if jsonOutput {
		return outputJSON(config)
	}

	fmt.Printf("Connected to Home Assistant\n\n")
	fmt.Printf("Version:       %s\n", config.Version)
	fmt.Printf("Location:      %s\n", config.LocationName)
	fmt.Printf("Time Zone:     %s\n", config.TimeZone)
	if config.State != "" {
		fmt.Printf("State:         %s\n", config.State)
	}
	if config.Country != "" {
		fmt.Printf("Country:       %s\n", config.Country)
	}
	if config.Language != "" {
		fmt.Printf("Language:      %s\n", config.Language)
	}
	fmt.Printf("Components:    %d loaded\n", len(config.Components))

	return nil
}
