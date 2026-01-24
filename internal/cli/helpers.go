package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var helpersCmd = &cobra.Command{
	Use:   "helpers",
	Short: "List and manage helpers",
	Long: `List and manage Home Assistant helpers (input entities).

Helpers are user-configurable entities like dropdowns, toggles, numbers, and text inputs.
Currently supports input_select (dropdown) helpers.

Examples:
  hass-cli helpers                        # List all helpers
  hass-cli helpers --json                 # Output as JSON
  hass-cli helpers inspect <helper_id>    # Show helper configuration
  hass-cli helpers create-select <name>   # Create a dropdown helper
  hass-cli helpers delete <helper_id>     # Delete a helper`,
	RunE: runHelpers,
}

var helpersInspectCmd = &cobra.Command{
	Use:   "inspect <helper_id>",
	Short: "Show detailed configuration of a helper",
	Long: `Show the helper state and attributes.

The helper_id is the full entity ID (e.g., 'input_select.my_dropdown').

Examples:
  hass-cli helpers inspect input_select.my_dropdown
  hass-cli helpers inspect input_boolean.my_toggle --json`,
	Args: cobra.ExactArgs(1),
	RunE: runHelpersInspect,
}

var helpersCreateSelectCmd = &cobra.Command{
	Use:   "create-select <name>",
	Short: "Create a new dropdown (input_select) helper",
	Long: `Create a new input_select helper with the specified name.

Options must be provided as a JSON array.

Examples:
  hass-cli helpers create-select "My Dropdown" --options '["option1","option2","option3"]'
  hass-cli helpers create-select "Room Scene" --options '["off","bright","dim"]' --icon mdi:lightbulb`,
	Args: cobra.ExactArgs(1),
	RunE: runHelpersCreateSelect,
}

var helpersEditSelectCmd = &cobra.Command{
	Use:   "edit-select <helper_id>",
	Short: "Edit an existing dropdown helper",
	Long: `Edit an existing input_select helper by updating its options.

Examples:
  hass-cli helpers edit-select input_select.my_dropdown --options '["new1","new2"]'`,
	Args: cobra.ExactArgs(1),
	RunE: runHelpersEditSelect,
}

var helpersRenameCmd = &cobra.Command{
	Use:   "rename <helper_id>",
	Short: "Rename a helper",
	Long: `Rename a helper's display name or entity ID.

Examples:
  hass-cli helpers rename input_button.my_button --name "Doorbell"
  hass-cli helpers rename input_button.my_button --new-id input_button.doorbell`,
	Args: cobra.ExactArgs(1),
	RunE: runHelpersRename,
}

var helpersDeleteCmd = &cobra.Command{
	Use:   "delete <helper_id>",
	Short: "Delete a helper",
	Long: `Delete a helper entity. This cannot be undone.

Examples:
  hass-cli helpers delete input_select.my_dropdown`,
	Args: cobra.ExactArgs(1),
	RunE: runHelpersDelete,
}

var helpersDisableCmd = &cobra.Command{
	Use:   "disable <helper_id>",
	Short: "Disable a helper",
	Long: `Disable a helper via the entity registry.

Examples:
  hass-cli helpers disable input_button.my_button`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelpersToggleDisabled(args[0], true)
	},
}

var helpersEnableCmd = &cobra.Command{
	Use:   "enable <helper_id>",
	Short: "Enable a helper",
	Long: `Enable (re-enable) a previously disabled helper.

Examples:
  hass-cli helpers enable input_button.my_button`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHelpersToggleDisabled(args[0], false)
	},
}

var (
	helperOptions     string
	helperIcon        string
	helperRenameName  string
	helperNewEntityID string
)

func init() {
	rootCmd.AddCommand(helpersCmd)

	helpersCmd.AddCommand(helpersInspectCmd)
	helpersCmd.AddCommand(helpersCreateSelectCmd)
	helpersCmd.AddCommand(helpersEditSelectCmd)
	helpersCmd.AddCommand(helpersRenameCmd)
	helpersCmd.AddCommand(helpersDeleteCmd)
	helpersCmd.AddCommand(helpersDisableCmd)
	helpersCmd.AddCommand(helpersEnableCmd)

	helpersCreateSelectCmd.Flags().StringVar(&helperOptions, "options", "", "JSON array of options (required)")
	helpersCreateSelectCmd.Flags().StringVar(&helperIcon, "icon", "", "Icon (e.g., mdi:lightbulb)")
	helpersCreateSelectCmd.MarkFlagRequired("options")

	helpersEditSelectCmd.Flags().StringVar(&helperOptions, "options", "", "JSON array of options")

	helpersRenameCmd.Flags().StringVar(&helperRenameName, "name", "", "New friendly name")
	helpersRenameCmd.Flags().StringVar(&helperNewEntityID, "new-id", "", "New entity ID (domain.object_id)")
}

type HelperInfo struct {
	EntityID     string   `json:"entity_id"`
	Name         string   `json:"name"`
	State        string   `json:"state"`
	Type         string   `json:"type"`
	Options      []string `json:"options,omitempty"`
	FriendlyName string   `json:"friendly_name,omitempty"`
}

func runHelpers(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	states, err := client.GetStates()
	if err != nil {
		return fmt.Errorf("failed to get states: %w", err)
	}

	var helpers []HelperInfo
	for _, state := range states {
		// Filter for input_* entities
		if !strings.HasPrefix(state.EntityID, "input_") {
			continue
		}

		helperType := strings.Split(state.EntityID, ".")[0]
		name := state.Attributes["friendly_name"]
		nameStr := ""
		if name != nil {
			nameStr = fmt.Sprintf("%v", name)
		}

		helper := HelperInfo{
			EntityID:     state.EntityID,
			Name:         nameStr,
			State:        state.State,
			Type:         helperType,
			FriendlyName: nameStr,
		}

		// Add options for input_select
		if helperType == "input_select" {
			if opts, ok := state.Attributes["options"].([]interface{}); ok {
				for _, opt := range opts {
					helper.Options = append(helper.Options, fmt.Sprintf("%v", opt))
				}
			}
		}

		helpers = append(helpers, helper)
	}

	// Sort by entity ID
	sort.Slice(helpers, func(i, j int) bool {
		return helpers[i].EntityID < helpers[j].EntityID
	})

	if jsonOutput {
		return outputJSON(helpers)
	}

	return outputHelpersTable(helpers)
}

func outputHelpersTable(helpers []HelperInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENTITY ID\tTYPE\tSTATE\tNAME")
	fmt.Fprintln(w, "---------\t----\t-----\t----")

	for _, h := range helpers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			h.EntityID,
			h.Type,
			h.State,
			h.Name,
		)
	}

	fmt.Fprintf(w, "\nTotal: %d helpers\n", len(helpers))
	w.Flush()
	return nil
}

func runHelpersInspect(cmd *cobra.Command, args []string) error {
	helperID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	state, err := client.GetState(helperID)
	if err != nil {
		return fmt.Errorf("failed to get helper state: %w", err)
	}

	return outputJSON(state)
}

func runHelpersCreateSelect(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Parse options
	var options []string
	if err := json.Unmarshal([]byte(helperOptions), &options); err != nil {
		return fmt.Errorf("invalid options JSON: %w", err)
	}

	if len(options) == 0 {
		return fmt.Errorf("at least one option is required")
	}

	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Home Assistant: %w", err)
	}
	defer wsClient.Close()

	helper, err := wsClient.CreateInputSelect(name, options, helperIcon)
	if err != nil {
		return fmt.Errorf("failed to create input_select: %w", err)
	}

	fmt.Printf("Input select created: %s\n", helper.Name)
	fmt.Printf("Entity ID: input_select.%s\n", helper.ID)
	fmt.Printf("\nNote: You may need to reload input_select or restart Home Assistant for the new helper to appear.\n")

	return nil
}

func runHelpersEditSelect(cmd *cobra.Command, args []string) error {
	helperID := args[0]

	if !strings.HasPrefix(helperID, "input_select.") {
		return fmt.Errorf("helper ID must be an input_select entity (e.g., input_select.my_dropdown)")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Parse options
	var options []string
	if err := json.Unmarshal([]byte(helperOptions), &options); err != nil {
		return fmt.Errorf("invalid options JSON: %w", err)
	}

	if len(options) == 0 {
		return fmt.Errorf("at least one option is required")
	}

	// Update options via service call
	err = client.CallInputSelectSetOptions(helperID, options)
	if err != nil {
		return fmt.Errorf("failed to update options: %w", err)
	}

	fmt.Printf("Input select updated: %s\n", helperID)
	return nil
}

func runHelpersDelete(cmd *cobra.Command, args []string) error {
	helperID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	domain, objectID, err := parseHelperID(helperID)
	if err != nil {
		return err
	}

	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Home Assistant: %w", err)
	}
	defer wsClient.Close()

	if err := wsClient.DeleteHelper(domain, objectID); err != nil {
		return fmt.Errorf("failed to delete helper: %w", err)
	}

	fmt.Printf("Helper deleted: %s\n", helperID)
	fmt.Printf("\nNote: You may need to reload %s or restart Home Assistant for the change to take effect.\n", domain)

	return nil
}

func runHelpersRename(cmd *cobra.Command, args []string) error {
	helperID := args[0]

	if helperRenameName == "" && helperNewEntityID == "" {
		return fmt.Errorf("must provide --name or --new-id")
	}

	domain, _, err := parseHelperID(helperID)
	if err != nil {
		return err
	}

	if helperNewEntityID != "" {
		newDomain, _, err := parseHelperID(helperNewEntityID)
		if err != nil {
			return fmt.Errorf("invalid new entity ID: %w", err)
		}
		if newDomain != domain {
			return fmt.Errorf("new entity ID must use the same domain (%s)", domain)
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Home Assistant: %w", err)
	}
	defer wsClient.Close()

	updates := make(map[string]interface{})
	if helperRenameName != "" {
		updates["name"] = helperRenameName
	}
	if helperNewEntityID != "" {
		updates["new_entity_id"] = helperNewEntityID
	}

	entity, err := wsClient.UpdateEntity(helperID, updates)
	if err != nil {
		return fmt.Errorf("failed to rename helper: %w", err)
	}

	if helperNewEntityID != "" && helperNewEntityID != helperID {
		fmt.Printf("Helper entity ID updated: %s -> %s\n", helperID, entity.EntityID)
	} else {
		fmt.Printf("Helper updated: %s\n", entity.EntityID)
	}

	if helperRenameName != "" {
		newName := helperRenameName
		if entity.Name != nil && *entity.Name != "" {
			newName = *entity.Name
		}
		fmt.Printf("New name: %s\n", newName)
	}

	return nil
}

func runHelpersToggleDisabled(helperID string, disable bool) error {
	if _, _, err := parseHelperID(helperID); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Home Assistant: %w", err)
	}
	defer wsClient.Close()

	var disabledBy interface{}
	if disable {
		disabledBy = "user"
	} else {
		disabledBy = nil
	}

	updates := map[string]interface{}{
		"disabled_by": disabledBy,
	}

	entity, err := wsClient.UpdateEntity(helperID, updates)
	if err != nil {
		action := "enable helper"
		if disable {
			action = "disable helper"
		}
		return fmt.Errorf("failed to %s: %w", action, err)
	}

	status := "enabled"
	if disable {
		status = "disabled"
	}

	fmt.Printf("Helper %s: %s\n", status, entity.EntityID)
	return nil
}

func parseHelperID(helperID string) (string, string, error) {
	parts := strings.Split(helperID, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid helper ID format (expected domain.object_id)")
	}

	domain := parts[0]
	if !strings.HasPrefix(domain, "input_") {
		return "", "", fmt.Errorf("not a helper entity (must start with input_)")
	}

	return domain, parts[1], nil
}
