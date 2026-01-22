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

var scriptsCmd = &cobra.Command{
	Use:   "scripts",
	Short: "List and manage scripts",
	Long: `List and manage Home Assistant scripts.

Scripts are sequences of actions that can be triggered manually or by automations.
Use 'hass-cli scripts run <script_id>' to execute a script.

Examples:
  hass-cli scripts                        # List all scripts
  hass-cli scripts --json                 # Output as JSON
  hass-cli scripts inspect <script_id>    # Show script configuration
  hass-cli scripts create <name>          # Create a new script
  hass-cli scripts run <script_id>        # Trigger a script
  hass-cli scripts debug <script_id>      # Show execution traces
  hass-cli scripts delete <script_id>     # Delete a script`,
	RunE: runScripts,
}

var scriptsInspectCmd = &cobra.Command{
	Use:   "inspect <script_id>",
	Short: "Show detailed configuration of a script",
	Long: `Show the script configuration including sequence of actions.

The script_id is the object_id portion of the entity (e.g., 'hello_world' for 'script.hello_world').

Examples:
  hass-cli scripts inspect hello_world
  hass-cli scripts inspect my_script --json`,
	Args: cobra.ExactArgs(1),
	RunE: runScriptsInspect,
}

var scriptsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new script",
	Long: `Create a new script with the specified name.

You can provide the script sequence as JSON via --sequence flag or stdin.
If no sequence is provided, an empty script is created.

Examples:
  hass-cli scripts create "Hello World" --description "A test script"
  hass-cli scripts create "Turn Off Lights" --sequence '[{"service":"light.turn_off","target":{"area_id":"living_room"}}]'
  hass-cli scripts create "My Script" --icon mdi:script --mode single`,
	Args: cobra.ExactArgs(1),
	RunE: runScriptsCreate,
}

var scriptsEditCmd = &cobra.Command{
	Use:   "edit <script_id>",
	Short: "Edit an existing script",
	Long: `Edit an existing script configuration.

Use flags to update specific properties, or provide a complete sequence via --sequence.

Examples:
  hass-cli scripts edit hello_world --alias "Hello World Updated"
  hass-cli scripts edit hello_world --description "Updated description"
  hass-cli scripts edit hello_world --sequence '[{"service":"light.turn_on"}]'`,
	Args: cobra.ExactArgs(1),
	RunE: runScriptsEdit,
}

var scriptsRenameCmd = &cobra.Command{
	Use:   "rename <script_id> <new_name>",
	Short: "Rename a script",
	Long: `Rename a script by updating its alias.

Examples:
  hass-cli scripts rename hello_world "Hello World Updated"`,
	Args: cobra.ExactArgs(2),
	RunE: runScriptsRename,
}

var scriptsRunCmd = &cobra.Command{
	Use:     "run <script_id>",
	Aliases: []string{"trigger"},
	Short:   "Run/trigger a script",
	Long: `Execute a script by calling its service.

The script_id is the object_id portion of the entity (e.g., 'hello_world' for 'script.hello_world').

Examples:
  hass-cli scripts run hello_world
  hass-cli scripts run my_script --data '{"variable1":"value1"}'`,
	Args: cobra.ExactArgs(1),
	RunE: runScriptsRun,
}

var scriptsDebugCmd = &cobra.Command{
	Use:   "debug <script_id>",
	Short: "Show execution traces for debugging",
	Long: `List and inspect execution traces for a script.

This shows the history of script executions with timing and step information.
Use --run-id to see details of a specific execution.

Examples:
  hass-cli scripts debug hello_world              # List all traces
  hass-cli scripts debug hello_world --run-id <id>  # Show specific trace`,
	Args: cobra.ExactArgs(1),
	RunE: runScriptsDebug,
}

var scriptsDeleteCmd = &cobra.Command{
	Use:   "delete <script_id>",
	Short: "Delete a script",
	Long: `Delete a script by its ID.

The script_id is the object_id portion of the entity (e.g., 'hello_world' for 'script.hello_world').

Examples:
  hass-cli scripts delete hello_world`,
	Args: cobra.ExactArgs(1),
	RunE: runScriptsDelete,
}

var (
	scriptDescription string
	scriptIcon        string
	scriptMode        string
	scriptSequence    string
	scriptAlias       string
	scriptRunData     string
	scriptRunID       string
)

func init() {
	rootCmd.AddCommand(scriptsCmd)
	scriptsCmd.AddCommand(scriptsInspectCmd)
	scriptsCmd.AddCommand(scriptsCreateCmd)
	scriptsCmd.AddCommand(scriptsEditCmd)
	scriptsCmd.AddCommand(scriptsRenameCmd)
	scriptsCmd.AddCommand(scriptsRunCmd)
	scriptsCmd.AddCommand(scriptsDebugCmd)
	scriptsCmd.AddCommand(scriptsDeleteCmd)

	// Create flags
	scriptsCreateCmd.Flags().StringVar(&scriptDescription, "description", "", "Description of the script")
	scriptsCreateCmd.Flags().StringVar(&scriptIcon, "icon", "", "Icon for the script (e.g., mdi:script)")
	scriptsCreateCmd.Flags().StringVar(&scriptMode, "mode", "single", "Script mode: single, restart, queued, parallel")
	scriptsCreateCmd.Flags().StringVar(&scriptSequence, "sequence", "", "JSON array of actions for the script sequence")

	// Edit flags
	scriptsEditCmd.Flags().StringVar(&scriptAlias, "alias", "", "New alias/name for the script")
	scriptsEditCmd.Flags().StringVar(&scriptDescription, "description", "", "New description")
	scriptsEditCmd.Flags().StringVar(&scriptIcon, "icon", "", "New icon")
	scriptsEditCmd.Flags().StringVar(&scriptMode, "mode", "", "New mode: single, restart, queued, parallel")
	scriptsEditCmd.Flags().StringVar(&scriptSequence, "sequence", "", "New JSON array of actions")

	// Run flags
	scriptsRunCmd.Flags().StringVar(&scriptRunData, "data", "", "JSON data to pass to the script")

	// Debug flags
	scriptsDebugCmd.Flags().StringVar(&scriptRunID, "run-id", "", "Specific run ID to inspect")
}

// ScriptInfo combines script entity info with config details.
type ScriptInfo struct {
	EntityID    string `json:"entity_id"`
	Name        string `json:"name"`
	State       string `json:"state"`
	Icon        string `json:"icon,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Description string `json:"description,omitempty"`
	LastTriggered string `json:"last_triggered,omitempty"`
}

func runScripts(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching scripts...")
	states, err := client.GetStates()
	if err != nil {
		return fmt.Errorf("failed to get states: %w", err)
	}

	// Filter for script entities
	var scripts []ScriptInfo
	for _, state := range states {
		if !strings.HasPrefix(state.EntityID, "script.") {
			continue
		}

		name := ""
		if fn, ok := state.Attributes["friendly_name"].(string); ok {
			name = fn
		}

		icon := ""
		if ic, ok := state.Attributes["icon"].(string); ok {
			icon = ic
		}

		mode := ""
		if m, ok := state.Attributes["mode"].(string); ok {
			mode = m
		}

		description := ""
		if d, ok := state.Attributes["description"].(string); ok {
			description = d
		}

		lastTriggered := ""
		if lt, ok := state.Attributes["last_triggered"].(string); ok {
			lastTriggered = lt
		}

		scripts = append(scripts, ScriptInfo{
			EntityID:      state.EntityID,
			Name:          name,
			State:         state.State,
			Icon:          icon,
			Mode:          mode,
			Description:   description,
			LastTriggered: lastTriggered,
		})
	}

	// Sort by name
	sort.Slice(scripts, func(i, j int) bool {
		return strings.ToLower(scripts[i].Name) < strings.ToLower(scripts[j].Name)
	})

	if jsonOutput {
		return outputJSON(scripts)
	}

	return outputScriptsTable(scripts)
}

func outputScriptsTable(scripts []ScriptInfo) error {
	if len(scripts) == 0 {
		fmt.Println("No scripts found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENTITY ID\tNAME\tSTATE\tMODE\tLAST TRIGGERED")
	fmt.Fprintln(w, "---------\t----\t-----\t----\t--------------")

	for _, s := range scripts {
		name := s.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		lastTriggered := s.LastTriggered
		if lastTriggered != "" {
			// Parse and format the timestamp
			if t, err := time.Parse(time.RFC3339, lastTriggered); err == nil {
				lastTriggered = t.Local().Format("2006-01-02 15:04:05")
			}
		} else {
			lastTriggered = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			s.EntityID,
			name,
			s.State,
			s.Mode,
			lastTriggered,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d scripts\n", len(scripts))

	return nil
}

func runScriptsInspect(cmd *cobra.Command, args []string) error {
	scriptID := normalizeScriptID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching script configuration...")
	config, err := client.GetScriptConfig(scriptID)
	if err != nil {
		// If not found as config ID, try as full entity ID
		if strings.HasPrefix(args[0], "script.") {
			state, stateErr := client.GetState(args[0])
			if stateErr == nil {
				return outputJSON(state)
			}
		}
		return fmt.Errorf("failed to get script: %w", err)
	}

	return outputJSON(config)
}

func runScriptsCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Parse sequence if provided
	var sequence []map[string]interface{}
	if scriptSequence != "" {
		if err := json.Unmarshal([]byte(scriptSequence), &sequence); err != nil {
			return fmt.Errorf("invalid sequence JSON: %w", err)
		}
	} else {
		// Create empty sequence with a placeholder
		sequence = []map[string]interface{}{}
	}

	// Generate script ID from name
	scriptID := slugify(name)

	config := &api.ScriptConfig{
		Alias:       name,
		Description: scriptDescription,
		Icon:        scriptIcon,
		Mode:        scriptMode,
		Sequence:    sequence,
	}

	if config.Mode == "" {
		config.Mode = "single"
	}

	printInfo("Creating script '%s'...", name)
	if err := client.CreateScript(scriptID, config); err != nil {
		return fmt.Errorf("failed to create script: %w", err)
	}

	fmt.Printf("Script created: %s\n", name)
	fmt.Printf("Entity ID: script.%s\n", scriptID)
	fmt.Println("\nNote: You may need to reload scripts or restart Home Assistant for the new script to appear.")

	return nil
}

func runScriptsEdit(cmd *cobra.Command, args []string) error {
	scriptID := normalizeScriptID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Get existing config
	printInfo("Fetching current script configuration...")
	config, err := client.GetScriptConfig(scriptID)
	if err != nil {
		return fmt.Errorf("failed to get script: %w", err)
	}

	// Apply updates
	if scriptAlias != "" {
		config.Alias = scriptAlias
	}
	if cmd.Flags().Changed("description") {
		config.Description = scriptDescription
	}
	if cmd.Flags().Changed("icon") {
		config.Icon = scriptIcon
	}
	if cmd.Flags().Changed("mode") {
		config.Mode = scriptMode
	}
	if scriptSequence != "" {
		var sequence []map[string]interface{}
		if err := json.Unmarshal([]byte(scriptSequence), &sequence); err != nil {
			return fmt.Errorf("invalid sequence JSON: %w", err)
		}
		config.Sequence = sequence
	}

	printInfo("Updating script...")
	if err := client.UpdateScript(scriptID, config); err != nil {
		return fmt.Errorf("failed to update script: %w", err)
	}

	fmt.Printf("Script updated: %s\n", config.Alias)

	return nil
}

func runScriptsRename(cmd *cobra.Command, args []string) error {
	scriptID := normalizeScriptID(args[0])
	newName := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Get existing config
	printInfo("Fetching current script configuration...")
	config, err := client.GetScriptConfig(scriptID)
	if err != nil {
		return fmt.Errorf("failed to get script: %w", err)
	}

	oldName := config.Alias
	config.Alias = newName

	printInfo("Renaming script...")
	if err := client.UpdateScript(scriptID, config); err != nil {
		return fmt.Errorf("failed to rename script: %w", err)
	}

	fmt.Printf("Script renamed: '%s' -> '%s'\n", oldName, newName)

	return nil
}

func runScriptsRun(cmd *cobra.Command, args []string) error {
	scriptID := normalizeScriptID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Parse data if provided
	var data map[string]interface{}
	if scriptRunData != "" {
		if err := json.Unmarshal([]byte(scriptRunData), &data); err != nil {
			return fmt.Errorf("invalid data JSON: %w", err)
		}
	}

	printInfo("Triggering script '%s'...", scriptID)
	_, err = client.CallService("script", scriptID, data)
	if err != nil {
		return fmt.Errorf("failed to trigger script: %w", err)
	}

	printSuccess("Script triggered: script.%s", scriptID)

	return nil
}

func runScriptsDebug(cmd *cobra.Command, args []string) error {
	scriptID := normalizeScriptID(args[0])

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

	if scriptRunID != "" {
		// Get specific trace
		printInfo("Fetching trace details...")
		trace, err := wsClient.GetTrace("script", scriptID, scriptRunID)
		if err != nil {
			return fmt.Errorf("failed to get trace: %w", err)
		}

		return outputJSON(trace)
	}

	// List all traces
	printInfo("Fetching traces for script '%s'...", scriptID)
	traces, err := wsClient.ListTraces("script", scriptID)
	if err != nil {
		return fmt.Errorf("failed to list traces: %w", err)
	}

	if jsonOutput {
		return outputJSON(traces)
	}

	return outputTracesTable(traces)
}

func outputTracesTable(traces []websocket.TraceSummary) error {
	if len(traces) == 0 {
		fmt.Println("No traces found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUN ID\tSTATE\tRESULT\tSTARTED\tDURATION")
	fmt.Fprintln(w, "------\t-----\t------\t-------\t--------")

	for _, t := range traces {
		runID := t.RunID
		if len(runID) > 16 {
			runID = runID[:16] + "..."
		}

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
			runID,
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

func runScriptsDelete(cmd *cobra.Command, args []string) error {
	scriptID := normalizeScriptID(args[0])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Deleting script '%s'...", scriptID)
	if err := client.DeleteScript(scriptID); err != nil {
		return fmt.Errorf("failed to delete script: %w", err)
	}

	printSuccess("Script deleted: %s", scriptID)
	fmt.Println("\nNote: You may need to reload scripts or restart Home Assistant for the change to take effect.")

	return nil
}

// normalizeScriptID extracts the script ID from an entity ID if needed.
func normalizeScriptID(input string) string {
	if strings.HasPrefix(input, "script.") {
		return strings.TrimPrefix(input, "script.")
	}
	return input
}
