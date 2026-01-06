package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Get or set entity states",
	Long: `Get or set entity states in Home Assistant.

Examples:
  hass-cli state get light.living_room       # Get entity state
  hass-cli state set light.living_room on    # Set entity state`,
}

var stateGetCmd = &cobra.Command{
	Use:   "get <entity_id>",
	Short: "Get the current state of an entity",
	Long: `Get the current state and attributes of an entity.

Examples:
  hass-cli state get light.living_room
  hass-cli state get sensor.temperature
  hass-cli state get light.living_room --json`,
	Args: cobra.ExactArgs(1),
	RunE: runStateGet,
}

var stateSetCmd = &cobra.Command{
	Use:   "set <entity_id> <state> [--attr key=value]...",
	Short: "Set the state of an entity",
	Long: `Set the state of an entity. This directly sets the state representation
in Home Assistant and does NOT communicate with the actual device.

To control a device (e.g., turn on a light), use 'hass-cli call' instead.

This command is useful for:
- Creating custom sensor entities
- Testing and debugging
- Setting states for template entities

Examples:
  hass-cli state set sensor.custom_value 42
  hass-cli state set sensor.custom_value 42 --attr unit_of_measurement=°C
  hass-cli state set input_text.note "Hello World"`,
	Args: cobra.ExactArgs(2),
	RunE: runStateSet,
}

var stateAttributes []string

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateGetCmd)
	stateCmd.AddCommand(stateSetCmd)

	stateSetCmd.Flags().StringArrayVar(&stateAttributes, "attr", []string{}, "Set attribute (key=value), can be specified multiple times")
}

func runStateGet(cmd *cobra.Command, args []string) error {
	entityID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching state for %s...", entityID)
	state, err := client.GetState(entityID)
	if err != nil {
		return fmt.Errorf("failed to get state: %w", err)
	}

	if jsonOutput {
		return outputJSON(state)
	}

	// Human-readable output
	fmt.Printf("Entity:        %s\n", state.EntityID)
	fmt.Printf("State:         %s\n", state.State)
	fmt.Printf("Last Changed:  %s\n", formatTime(state.LastChanged))
	fmt.Printf("Last Updated:  %s\n", formatTime(state.LastUpdated))

	if len(state.Attributes) > 0 {
		fmt.Println("\nAttributes:")
		for key, value := range state.Attributes {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

func runStateSet(cmd *cobra.Command, args []string) error {
	entityID := args[0]
	newState := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Parse attributes
	var attrs map[string]interface{}
	if len(stateAttributes) > 0 {
		attrs = make(map[string]interface{})
		for _, attr := range stateAttributes {
			parts := strings.SplitN(attr, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid attribute format: %s (expected key=value)", attr)
			}
			key := parts[0]
			value := parts[1]

			// Try to parse as JSON for complex values
			var jsonValue interface{}
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				attrs[key] = jsonValue
			} else {
				attrs[key] = value
			}
		}
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Setting state for %s to %s...", entityID, newState)
	state, err := client.SetState(entityID, newState, attrs)
	if err != nil {
		return fmt.Errorf("failed to set state: %w", err)
	}

	if jsonOutput {
		return outputJSON(state)
	}

	fmt.Printf("State set successfully\n")
	fmt.Printf("Entity:        %s\n", state.EntityID)
	fmt.Printf("State:         %s\n", state.State)

	return nil
}

// formatTime formats an ISO timestamp for display.
func formatTime(timestamp string) string {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
