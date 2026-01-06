package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/dorinclisu/hass-cli/internal/config"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure Home Assistant server connection",
	Long: `Configure the Home Assistant server URL and access token.

You can provide the URL and token as flags, or you will be prompted to enter them.

To obtain a long-lived access token:
  1. Log into your Home Assistant web interface
  2. Go to your profile (click your username in the sidebar)
  3. Scroll down to "Long-Lived Access Tokens"
  4. Click "Create Token" and give it a name
  5. Copy the token (it will only be shown once)

Example:
  hass-cli login --url http://homeassistant.local:8123 --token YOUR_TOKEN`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Get URL and token from flags or prompt
	url := serverURL
	tkn := token

	reader := bufio.NewReader(os.Stdin)

	// Prompt for URL if not provided
	if url == "" {
		fmt.Print("Home Assistant URL (e.g., http://homeassistant.local:8123): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read URL: %w", err)
		}
		url = strings.TrimSpace(input)
	}

	// Validate URL
	if url == "" {
		return fmt.Errorf("URL is required")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// Prompt for token if not provided
	if tkn == "" {
		fmt.Print("Long-lived access token: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}
		tkn = strings.TrimSpace(input)
	}

	// Validate token
	if tkn == "" {
		return fmt.Errorf("token is required")
	}

	// Test the connection
	printInfo("Testing connection to %s...", url)
	client := api.NewClient(url, tkn, time.Duration(timeout)*time.Second)
	if err := client.CheckConnection(); err != nil {
		if api.IsUnauthorized(err) {
			return fmt.Errorf("authentication failed: invalid token")
		}
		return fmt.Errorf("connection failed: %w", err)
	}

	// Save the configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			URL:   url,
			Token: tkn,
		},
		Defaults: config.DefaultsConfig{
			Output:  "human",
			Timeout: timeout,
		},
	}

	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	if err := cfg.SaveTo(cfgPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	printSuccess("Successfully logged in to %s", url)
	printSuccess("Configuration saved to %s", cfgPath)

	return nil
}
