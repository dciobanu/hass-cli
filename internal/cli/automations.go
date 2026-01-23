package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var automationsCmd = &cobra.Command{
	Use:   "automations",
	Short: "List and manage automations",
	Long: `List and manage Home Assistant automations.

Automations are rules that trigger actions based on events, states, or time.
Use 'hass-cli automations trigger <automation_id>' to manually run an automation.

Examples:
  hass-cli automations                           # List all automations
  hass-cli automations --json                    # Output as JSON
  hass-cli automations inspect <automation_id>   # Show automation configuration
  hass-cli automations create <name>             # Create a new automation
  hass-cli automations trigger <automation_id>   # Manually trigger an automation
  hass-cli automations debug <automation_id>     # Show execution traces
  hass-cli automations delete <automation_id>    # Delete an automation`,
	RunE: runAutomations,
}

var automationsInspectCmd = &cobra.Command{
	Use:   "inspect <automation_id>",
	Short: "Show detailed configuration of an automation",
	Long: `Show the automation configuration including triggers, conditions, and actions.

The automation_id is the numeric ID of the automation configuration.
You can find automation IDs by running 'hass-cli automations --json'.

Examples:
  hass-cli automations inspect 1761025981191
  hass-cli automations inspect automation.brightness_change --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsInspect,
}

var automationsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new automation",
	Long: `Create a new automation with the specified name.

You can provide triggers, conditions, and actions as JSON via flags.
If no triggers/actions are provided, an empty automation is created.

Examples:
  hass-cli automations create "Motion Light" --description "Turn on light when motion detected"
  hass-cli automations create "Sunrise Routine" --triggers '[{"trigger":"sun","event":"sunrise"}]' --actions '[{"action":"light.turn_on","target":{"area_id":"bedroom"}}]'
  hass-cli automations create "Daily Backup" --mode single`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsCreate,
}

var automationsEditCmd = &cobra.Command{
	Use:   "edit <automation_id>",
	Short: "Edit an existing automation",
	Long: `Edit an existing automation configuration.

Use flags to update specific properties, or provide complete triggers/conditions/actions via JSON.

Examples:
  hass-cli automations edit 1761025981191 --alias "Updated Name"
  hass-cli automations edit 1761025981191 --description "New description"
  hass-cli automations edit 1761025981191 --actions '[{"action":"light.turn_off"}]'`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsEdit,
}

var automationsRenameCmd = &cobra.Command{
	Use:   "rename <automation_id> <new_name>",
	Short: "Rename an automation",
	Long: `Rename an automation by updating its alias.

Examples:
  hass-cli automations rename 1761025981191 "New Automation Name"`,
	Args: cobra.ExactArgs(2),
	RunE: runAutomationsRename,
}

var automationsTriggerCmd = &cobra.Command{
	Use:     "trigger <automation_id>",
	Aliases: []string{"run"},
	Short:   "Manually trigger an automation",
	Long: `Manually trigger an automation to run immediately.

The automation_id can be the numeric config ID or the entity ID.

Examples:
  hass-cli automations trigger 1761025981191
  hass-cli automations trigger automation.brightness_change
  hass-cli automations run brightness_change`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsTrigger,
}

var automationsDebugCmd = &cobra.Command{
	Use:   "debug <automation_id>",
	Short: "Show execution traces for debugging",
	Long: `List and inspect execution traces for an automation.

This shows the history of automation executions with timing and step information.
Use --run-id to see details of a specific execution.

Examples:
  hass-cli automations debug 1761025981191              # List all traces
  hass-cli automations debug 1761025981191 --run-id <id>  # Show specific trace`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsDebug,
}

var automationsDeleteCmd = &cobra.Command{
	Use:   "delete <automation_id>",
	Short: "Delete an automation",
	Long: `Delete an automation by its configuration ID.

The automation_id is the numeric ID of the automation configuration.
You can find automation IDs by running 'hass-cli automations --json'.

Examples:
  hass-cli automations delete 1761025981191`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsDelete,
}

var automationsEnableCmd = &cobra.Command{
	Use:   "enable <automation_id>",
	Short: "Enable a disabled automation",
	Long: `Enable an automation that was previously disabled.

Examples:
  hass-cli automations enable automation.brightness_change
  hass-cli automations enable brightness_change`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsEnable,
}

var automationsDisableCmd = &cobra.Command{
	Use:   "disable <automation_id>",
	Short: "Disable an automation",
	Long: `Disable an automation so it won't run automatically.

The automation can still be triggered manually.

Examples:
  hass-cli automations disable automation.brightness_change
  hass-cli automations disable brightness_change`,
	Args: cobra.ExactArgs(1),
	RunE: runAutomationsDisable,
}

var (
	automationDescription string
	automationMode        string
	automationTriggers    string
	automationConditions  string
	automationActions     string
	automationAlias       string
	automationRunID       string
)

func init() {
	rootCmd.AddCommand(automationsCmd)
	automationsCmd.AddCommand(automationsInspectCmd)
	automationsCmd.AddCommand(automationsCreateCmd)
	automationsCmd.AddCommand(automationsEditCmd)
	automationsCmd.AddCommand(automationsRenameCmd)
	automationsCmd.AddCommand(automationsTriggerCmd)
	automationsCmd.AddCommand(automationsDebugCmd)
	automationsCmd.AddCommand(automationsDeleteCmd)
	automationsCmd.AddCommand(automationsEnableCmd)
	automationsCmd.AddCommand(automationsDisableCmd)

	// Create flags
	automationsCreateCmd.Flags().StringVar(&automationDescription, "description", "", "Description of the automation")
	automationsCreateCmd.Flags().StringVar(&automationMode, "mode", "single", "Automation mode: single, restart, queued, parallel")
	automationsCreateCmd.Flags().StringVar(&automationTriggers, "triggers", "", "JSON array of triggers")
	automationsCreateCmd.Flags().StringVar(&automationConditions, "conditions", "", "JSON array of conditions")
	automationsCreateCmd.Flags().StringVar(&automationActions, "actions", "", "JSON array of actions")

	// Edit flags
	automationsEditCmd.Flags().StringVar(&automationAlias, "alias", "", "New alias/name for the automation")
	automationsEditCmd.Flags().StringVar(&automationDescription, "description", "", "New description")
	automationsEditCmd.Flags().StringVar(&automationMode, "mode", "", "New mode: single, restart, queued, parallel")
	automationsEditCmd.Flags().StringVar(&automationTriggers, "triggers", "", "New JSON array of triggers")
	automationsEditCmd.Flags().StringVar(&automationConditions, "conditions", "", "New JSON array of conditions")
	automationsEditCmd.Flags().StringVar(&automationActions, "actions", "", "New JSON array of actions")

	// Debug flags
	automationsDebugCmd.Flags().StringVar(&automationRunID, "run-id", "", "Specific run ID to inspect")
}

// AutomationInfo combines automation entity info with config details.
type AutomationInfo struct {
	EntityID      string `json:"entity_id"`
	Name          string `json:"name"`
	State         string `json:"state"`
	ConfigID      string `json:"config_id,omitempty"`
	Mode          string `json:"mode,omitempty"`
	LastTriggered string `json:"last_triggered,omitempty"`
	CurrentRuns   int    `json:"current,omitempty"`
}

func runAutomations(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching automations...")
	states, err := client.GetStates()
	if err != nil {
		return fmt.Errorf("failed to get states: %w", err)
	}

	// Filter for automation entities
	var automations []AutomationInfo
	for _, state := range states {
		if !strings.HasPrefix(state.EntityID, "automation.") {
			continue
		}

		name := ""
		if fn, ok := state.Attributes["friendly_name"].(string); ok {
			name = fn
		}

		mode := ""
		if m, ok := state.Attributes["mode"].(string); ok {
			mode = m
		}

		configID := ""
		if id, ok := state.Attributes["id"].(string); ok {
			configID = id
		} else if id, ok := state.Attributes["id"].(float64); ok {
			configID = strconv.FormatFloat(id, 'f', 0, 64)
		}

		lastTriggered := ""
		if lt, ok := state.Attributes["last_triggered"].(string); ok {
			lastTriggered = lt
		}

		currentRuns := 0
		if cur, ok := state.Attributes["current"].(float64); ok {
			currentRuns = int(cur)
		}

		automations = append(automations, AutomationInfo{
			EntityID:      state.EntityID,
			Name:          name,
			State:         state.State,
			ConfigID:      configID,
			Mode:          mode,
			LastTriggered: lastTriggered,
			CurrentRuns:   currentRuns,
		})
	}

	// Sort by name
	sort.Slice(automations, func(i, j int) bool {
		return strings.ToLower(automations[i].Name) < strings.ToLower(automations[j].Name)
	})

	if jsonOutput {
		return outputJSON(automations)
	}

	return outputAutomationsTable(automations)
}

func outputAutomationsTable(automations []AutomationInfo) error {
	if len(automations) == 0 {
		fmt.Println("No automations found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CONFIG ID\tNAME\tSTATE\tMODE\tLAST TRIGGERED")
	fmt.Fprintln(w, "---------\t----\t-----\t----\t--------------")

	for _, a := range automations {
		name := a.Name
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		configID := a.ConfigID
		if configID == "" {
			configID = "-"
		}

		lastTriggered := a.LastTriggered
		if lastTriggered != "" && lastTriggered != "None" {
			// Parse and format the timestamp
			if t, err := time.Parse(time.RFC3339, lastTriggered); err == nil {
				lastTriggered = t.Local().Format("2006-01-02 15:04:05")
			}
		} else {
			lastTriggered = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			configID,
			name,
			a.State,
			a.Mode,
			lastTriggered,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d automations\n", len(automations))

	return nil
}

func runAutomationsInspect(cmd *cobra.Command, args []string) error {
	automationID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Try to extract config ID from entity ID if needed
	configID := normalizeAutomationID(automationID)

	// If it looks like an entity ID, we need to look up the config ID
	if strings.HasPrefix(automationID, "automation.") {
		printInfo("Looking up automation config ID...")
		state, err := client.GetState(automationID)
		if err != nil {
			return fmt.Errorf("failed to get automation state: %w", err)
		}
		if id, ok := state.Attributes["id"].(string); ok {
			configID = id
		} else if id, ok := state.Attributes["id"].(float64); ok {
			configID = strconv.FormatFloat(id, 'f', 0, 64)
		} else {
			return fmt.Errorf("could not find config ID for %s", automationID)
		}
	}

	printInfo("Fetching automation configuration...")
	config, err := client.GetAutomationConfig(configID)
	if err != nil {
		return fmt.Errorf("failed to get automation: %w", err)
	}

	return outputJSON(config)
}

func runAutomationsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Parse triggers if provided
	var triggers []map[string]interface{}
	if automationTriggers != "" {
		if err := json.Unmarshal([]byte(automationTriggers), &triggers); err != nil {
			return fmt.Errorf("invalid triggers JSON: %w", err)
		}
	} else {
		triggers = []map[string]interface{}{}
	}

	// Parse conditions if provided
	var conditions []map[string]interface{}
	if automationConditions != "" {
		if err := json.Unmarshal([]byte(automationConditions), &conditions); err != nil {
			return fmt.Errorf("invalid conditions JSON: %w", err)
		}
	} else {
		conditions = []map[string]interface{}{}
	}

	// Parse actions if provided
	var actions []map[string]interface{}
	if automationActions != "" {
		if err := json.Unmarshal([]byte(automationActions), &actions); err != nil {
			return fmt.Errorf("invalid actions JSON: %w", err)
		}
	} else {
		actions = []map[string]interface{}{}
	}

	// Generate automation ID from timestamp
	automationID := strconv.FormatInt(time.Now().UnixMilli(), 10)

	config := &api.AutomationConfig{
		ID:          automationID,
		Alias:       name,
		Description: automationDescription,
		Mode:        automationMode,
		Triggers:    triggers,
		Conditions:  conditions,
		Actions:     actions,
	}

	if config.Mode == "" {
		config.Mode = "single"
	}

	printInfo("Creating automation '%s'...", name)
	if err := client.CreateAutomation(automationID, config); err != nil {
		return fmt.Errorf("failed to create automation: %w", err)
	}

	fmt.Printf("Automation created: %s\n", name)
	fmt.Printf("Config ID: %s\n", automationID)
	fmt.Printf("Entity ID will be: automation.%s\n", slugify(name))
	fmt.Println("\nNote: You may need to reload automations or restart Home Assistant for the new automation to appear.")

	return nil
}

func runAutomationsEdit(cmd *cobra.Command, args []string) error {
	automationID := normalizeAutomationID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Get existing config
	printInfo("Fetching current automation configuration...")
	config, err := client.GetAutomationConfig(automationID)
	if err != nil {
		return fmt.Errorf("failed to get automation: %w", err)
	}

	// Apply updates
	if automationAlias != "" {
		config.Alias = automationAlias
	}
	if cmd.Flags().Changed("description") {
		config.Description = automationDescription
	}
	if cmd.Flags().Changed("mode") {
		config.Mode = automationMode
	}
	if automationTriggers != "" {
		var triggers []map[string]interface{}
		if err := json.Unmarshal([]byte(automationTriggers), &triggers); err != nil {
			return fmt.Errorf("invalid triggers JSON: %w", err)
		}
		config.Triggers = triggers
	}
	if automationConditions != "" {
		var conditions []map[string]interface{}
		if err := json.Unmarshal([]byte(automationConditions), &conditions); err != nil {
			return fmt.Errorf("invalid conditions JSON: %w", err)
		}
		config.Conditions = conditions
	}
	if automationActions != "" {
		var actions []map[string]interface{}
		if err := json.Unmarshal([]byte(automationActions), &actions); err != nil {
			return fmt.Errorf("invalid actions JSON: %w", err)
		}
		config.Actions = actions
	}

	printInfo("Updating automation...")
	if err := client.UpdateAutomation(automationID, config); err != nil {
		return fmt.Errorf("failed to update automation: %w", err)
	}

	fmt.Printf("Automation updated: %s\n", config.Alias)

	return nil
}

func runAutomationsRename(cmd *cobra.Command, args []string) error {
	automationID := normalizeAutomationID(args[0])
	newName := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Get existing config
	printInfo("Fetching current automation configuration...")
	config, err := client.GetAutomationConfig(automationID)
	if err != nil {
		return fmt.Errorf("failed to get automation: %w", err)
	}

	oldName := config.Alias
	config.Alias = newName

	printInfo("Renaming automation...")
	if err := client.UpdateAutomation(automationID, config); err != nil {
		return fmt.Errorf("failed to rename automation: %w", err)
	}

	fmt.Printf("Automation renamed: '%s' -> '%s'\n", oldName, newName)

	return nil
}

func runAutomationsTrigger(cmd *cobra.Command, args []string) error {
	automationID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Build entity ID if needed
	entityID := automationID
	if !strings.HasPrefix(entityID, "automation.") {
		// Check if it's a config ID or object ID
		if _, err := strconv.ParseInt(automationID, 10, 64); err == nil {
			// It's a numeric config ID, need to find the entity
			states, err := client.GetStates()
			if err != nil {
				return fmt.Errorf("failed to get states: %w", err)
			}
			for _, state := range states {
				if !strings.HasPrefix(state.EntityID, "automation.") {
					continue
				}
				if id, ok := state.Attributes["id"].(string); ok && id == automationID {
					entityID = state.EntityID
					break
				} else if id, ok := state.Attributes["id"].(float64); ok {
					if strconv.FormatFloat(id, 'f', 0, 64) == automationID {
						entityID = state.EntityID
						break
					}
				}
			}
		} else {
			entityID = "automation." + automationID
		}
	}

	printInfo("Triggering automation '%s'...", entityID)
	_, err = client.CallService("automation", "trigger", map[string]interface{}{
		"entity_id": entityID,
	})
	if err != nil {
		return fmt.Errorf("failed to trigger automation: %w", err)
	}

	printSuccess("Automation triggered: %s", entityID)

	return nil
}

func runAutomationsDebug(cmd *cobra.Command, args []string) error {
	automationID := normalizeAutomationID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	printInfo("Connecting to Home Assistant...")
	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	if automationRunID != "" {
		// Get specific trace
		printInfo("Fetching trace details...")
		trace, err := wsClient.GetTrace("automation", automationID, automationRunID)
		if err != nil {
			return fmt.Errorf("failed to get trace: %w", err)
		}

		return outputJSON(trace)
	}

	// List all traces
	printInfo("Fetching traces for automation '%s'...", automationID)
	traces, err := wsClient.ListTraces("automation", automationID)
	if err != nil {
		return fmt.Errorf("failed to list traces: %w", err)
	}

	if jsonOutput {
		return outputJSON(traces)
	}

	return outputAutomationTracesTable(traces)
}

func outputAutomationTracesTable(traces []websocket.TraceSummary) error {
	if len(traces) == 0 {
		fmt.Println("No traces found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUN ID\tSTATE\tRESULT\tSTARTED\tDURATION")
	fmt.Fprintln(w, "------\t-----\t------\t-------\t--------")

	for _, t := range traces {
		started := t.Timestamp.Start
		if s, err := time.Parse(time.RFC3339, t.Timestamp.Start); err == nil {
			started = s.Local().Format("2006-01-02 15:04:05")
		}

		duration := ""
		if t.Timestamp.Start != "" && t.Timestamp.Finish != "" {
			start, err1 := time.Parse(time.RFC3339, t.Timestamp.Start)
			finish, err2 := time.Parse(time.RFC3339, t.Timestamp.Finish)
			if err1 == nil && err2 == nil {
				d := finish.Sub(start)
				if d < time.Second {
					duration = fmt.Sprintf("%dms", d.Milliseconds())
				} else {
					duration = d.Round(time.Millisecond).String()
				}
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			t.RunID,
			t.State,
			t.ScriptExecution,
			started,
			duration,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d traces\n", len(traces))
	fmt.Println("\nUse --run-id <id> to see detailed trace information")

	return nil
}

func runAutomationsDelete(cmd *cobra.Command, args []string) error {
	automationID := normalizeAutomationID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Deleting automation '%s'...", automationID)
	if err := client.DeleteAutomation(automationID); err != nil {
		return fmt.Errorf("failed to delete automation: %w", err)
	}

	printSuccess("Automation deleted: %s", automationID)
	fmt.Println("\nNote: You may need to reload automations or restart Home Assistant for the change to take effect.")

	return nil
}

func runAutomationsEnable(cmd *cobra.Command, args []string) error {
	automationID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Build entity ID if needed
	entityID := buildAutomationEntityID(automationID, client)

	printInfo("Enabling automation '%s'...", entityID)
	_, err = client.CallService("automation", "turn_on", map[string]interface{}{
		"entity_id": entityID,
	})
	if err != nil {
		return fmt.Errorf("failed to enable automation: %w", err)
	}

	printSuccess("Automation enabled: %s", entityID)

	return nil
}

func runAutomationsDisable(cmd *cobra.Command, args []string) error {
	automationID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Build entity ID if needed
	entityID := buildAutomationEntityID(automationID, client)

	printInfo("Disabling automation '%s'...", entityID)
	_, err = client.CallService("automation", "turn_off", map[string]interface{}{
		"entity_id": entityID,
	})
	if err != nil {
		return fmt.Errorf("failed to disable automation: %w", err)
	}

	printSuccess("Automation disabled: %s", entityID)

	return nil
}

// normalizeAutomationID extracts the automation config ID from various input formats.
func normalizeAutomationID(input string) string {
	// Remove automation. prefix if present
	if strings.HasPrefix(input, "automation.") {
		return strings.TrimPrefix(input, "automation.")
	}
	return input
}

// buildAutomationEntityID converts a config ID or object ID to a full entity ID.
func buildAutomationEntityID(automationID string, client *api.Client) string {
	if strings.HasPrefix(automationID, "automation.") {
		return automationID
	}

	// Check if it's a numeric config ID
	if _, err := strconv.ParseInt(automationID, 10, 64); err == nil {
		// It's a numeric config ID, need to find the entity
		states, err := client.GetStates()
		if err == nil {
			for _, state := range states {
				if !strings.HasPrefix(state.EntityID, "automation.") {
					continue
				}
				if id, ok := state.Attributes["id"].(string); ok && id == automationID {
					return state.EntityID
				} else if id, ok := state.Attributes["id"].(float64); ok {
					if strconv.FormatFloat(id, 'f', 0, 64) == automationID {
						return state.EntityID
					}
				}
			}
		}
	}

	// Assume it's an object ID
	return "automation." + automationID
}
