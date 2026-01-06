package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/spf13/cobra"
)

var callCmd = &cobra.Command{
	Use:   "call <domain.service>",
	Short: "Call a Home Assistant service",
	Long: `Call a service in Home Assistant.

This is the primary way to control devices. Use 'hass-cli services' to list
available services and 'hass-cli services inspect' to see service details.

Examples:
  hass-cli call light.turn_on -e light.living_room
  hass-cli call light.turn_off -a living_room
  hass-cli call light.turn_on -a kitchen --data '{"brightness": 128}'
  hass-cli call switch.toggle -e switch.fan
  hass-cli call scene.turn_on -e scene.movie_night
  hass-cli call homeassistant.restart
  hass-cli call notify.mobile_app --data '{"message": "Hello!"}'`,
	Args: cobra.ExactArgs(1),
	RunE: runCall,
}

var (
	callEntityID string
	callAreaID   string
	callData     string
	callDataArgs []string
)

func init() {
	rootCmd.AddCommand(callCmd)

	callCmd.Flags().StringVarP(&callEntityID, "entity", "e", "", "Target entity ID")
	callCmd.Flags().StringVarP(&callAreaID, "area", "a", "", "Target area ID")
	callCmd.Flags().StringVar(&callData, "data", "", "Service data as JSON string")
	callCmd.Flags().StringArrayVarP(&callDataArgs, "set", "s", []string{}, "Set service data field (key=value), can be specified multiple times")
}

func runCall(cmd *cobra.Command, args []string) error {
	fullService := args[0]

	parts := strings.SplitN(fullService, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid service format: %s (expected domain.service)", fullService)
	}
	domain := parts[0]
	service := parts[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Build service data
	data := make(map[string]interface{})

	// Add entity_id if specified
	if callEntityID != "" {
		data["entity_id"] = callEntityID
	}

	// Add area_id if specified
	if callAreaID != "" {
		data["area_id"] = callAreaID
	}

	// Parse --data JSON if provided
	if callData != "" {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(callData), &jsonData); err != nil {
			return fmt.Errorf("invalid JSON in --data: %w", err)
		}
		for k, v := range jsonData {
			data[k] = v
		}
	}

	// Parse --set arguments
	for _, arg := range callDataArgs {
		keyValue := strings.SplitN(arg, "=", 2)
		if len(keyValue) != 2 {
			return fmt.Errorf("invalid --set format: %s (expected key=value)", arg)
		}
		key := keyValue[0]
		value := keyValue[1]

		// Try to parse as JSON for complex values (numbers, booleans, arrays, objects)
		var jsonValue interface{}
		if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
			data[key] = jsonValue
		} else {
			data[key] = value
		}
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Calling %s.%s...", domain, service)
	changedStates, err := client.CallService(domain, service, data)
	if err != nil {
		return fmt.Errorf("service call failed: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"success":        true,
			"changed_states": changedStates,
		})
	}

	fmt.Printf("Service %s.%s called successfully\n", domain, service)

	if len(changedStates) > 0 {
		fmt.Printf("\nChanged states (%d):\n", len(changedStates))
		for _, state := range changedStates {
			fmt.Printf("  %s: %s\n", state.EntityID, state.State)
		}
	}

	return nil
}
