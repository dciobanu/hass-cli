package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/spf13/cobra"
)

var scenesCmd = &cobra.Command{
	Use:   "scenes",
	Short: "List and manage scenes",
	Long: `List and manage Home Assistant scenes.

Scenes capture the state of multiple entities and can be activated to restore
those states. Use 'hass-cli call scene.turn_on -e scene.<name>' to activate.

Examples:
  hass-cli scenes                        # List all scenes
  hass-cli scenes --json                 # Output as JSON
  hass-cli scenes inspect <scene_id>     # Show scene configuration
  hass-cli scenes create "Movie Night"   # Create scene from current states
  hass-cli scenes delete <scene_id>      # Delete a scene`,
	RunE: runScenes,
}

var scenesInspectCmd = &cobra.Command{
	Use:   "inspect <scene_id>",
	Short: "Show detailed information about a scene",
	Long: `Show the scene configuration including all entities and their states.

The scene_id is the numeric ID of the scene configuration, not the entity ID.
You can find scene IDs by running 'hass-cli scenes --json'.

Examples:
  hass-cli scenes inspect 1767672291452`,
	Args: cobra.ExactArgs(1),
	RunE: runScenesInspect,
}

var scenesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new scene",
	Long: `Create a new scene capturing the current state of specified entities.

If no entities are specified, you must provide them via --entity flags.
The scene will capture the current state of each entity.

Examples:
  hass-cli scenes create "Movie Night" -e light.living_room -e light.kitchen
  hass-cli scenes create "Good Morning" -e light.bedroom --icon mdi:weather-sunny`,
	Args: cobra.ExactArgs(1),
	RunE: runScenesCreate,
}

var scenesDeleteCmd = &cobra.Command{
	Use:   "delete <scene_id>",
	Short: "Delete a scene",
	Long: `Delete a scene by its configuration ID.

The scene_id is the numeric ID of the scene configuration.
You can find scene IDs by running 'hass-cli scenes --json'.

Examples:
  hass-cli scenes delete 1767672291452`,
	Args: cobra.ExactArgs(1),
	RunE: runScenesDelete,
}

var scenesAddEntityCmd = &cobra.Command{
	Use:   "add-entity <scene_id> <entity_id>",
	Short: "Add an entity to a scene",
	Long: `Add an entity to an existing scene, capturing its current state.

Examples:
  hass-cli scenes add-entity 1767672291452 light.kitchen`,
	Args: cobra.ExactArgs(2),
	RunE: runScenesAddEntity,
}

var scenesRemoveEntityCmd = &cobra.Command{
	Use:   "remove-entity <scene_id> <entity_id>",
	Short: "Remove an entity from a scene",
	Long: `Remove an entity from an existing scene.

Examples:
  hass-cli scenes remove-entity 1767672291452 light.kitchen`,
	Args: cobra.ExactArgs(2),
	RunE: runScenesRemoveEntity,
}

var (
	sceneEntities []string
	sceneIcon     string
)

func init() {
	rootCmd.AddCommand(scenesCmd)
	scenesCmd.AddCommand(scenesInspectCmd)
	scenesCmd.AddCommand(scenesCreateCmd)
	scenesCmd.AddCommand(scenesDeleteCmd)
	scenesCmd.AddCommand(scenesAddEntityCmd)
	scenesCmd.AddCommand(scenesRemoveEntityCmd)

	scenesCreateCmd.Flags().StringArrayVarP(&sceneEntities, "entity", "e", []string{}, "Entity to include in scene (can be specified multiple times)")
	scenesCreateCmd.Flags().StringVar(&sceneIcon, "icon", "", "Icon for the scene (e.g., mdi:movie)")
}

// SceneInfo combines scene entity info with config details.
type SceneInfo struct {
	EntityID   string                 `json:"entity_id"`
	Name       string                 `json:"name"`
	State      string                 `json:"state"`
	Icon       string                 `json:"icon,omitempty"`
	ConfigID   string                 `json:"config_id,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func runScenes(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching scenes...")
	states, err := client.GetStates()
	if err != nil {
		return fmt.Errorf("failed to get states: %w", err)
	}

	// Filter for scene entities
	var scenes []SceneInfo
	for _, state := range states {
		if !strings.HasPrefix(state.EntityID, "scene.") {
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

		// Try to get config ID from attributes
		configID := ""
		if id, ok := state.Attributes["id"].(string); ok {
			configID = id
		} else if id, ok := state.Attributes["id"].(float64); ok {
			configID = strconv.FormatFloat(id, 'f', 0, 64)
		}

		scenes = append(scenes, SceneInfo{
			EntityID:   state.EntityID,
			Name:       name,
			State:      state.State,
			Icon:       icon,
			ConfigID:   configID,
			Attributes: state.Attributes,
		})
	}

	// Sort by name
	sort.Slice(scenes, func(i, j int) bool {
		return strings.ToLower(scenes[i].Name) < strings.ToLower(scenes[j].Name)
	})

	if jsonOutput {
		return outputJSON(scenes)
	}

	return outputScenesTable(scenes)
}

func outputScenesTable(scenes []SceneInfo) error {
	if len(scenes) == 0 {
		fmt.Println("No scenes found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENTITY ID\tNAME\tCONFIG ID\tICON")
	fmt.Fprintln(w, "---------\t----\t---------\t----")

	for _, s := range scenes {
		name := s.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		configID := s.ConfigID
		if configID == "" {
			configID = "-"
		}

		icon := s.Icon
		if icon == "" {
			icon = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			s.EntityID,
			name,
			configID,
			icon,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d scenes\n", len(scenes))

	return nil
}

func runScenesInspect(cmd *cobra.Command, args []string) error {
	sceneID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching scene configuration...")
	config, err := client.GetSceneConfig(sceneID)
	if err != nil {
		// If not found as config ID, try as entity ID
		if strings.HasPrefix(sceneID, "scene.") {
			state, stateErr := client.GetState(sceneID)
			if stateErr == nil {
				return outputJSON(state)
			}
		}
		return fmt.Errorf("failed to get scene: %w", err)
	}

	return outputJSON(config)
}

func runScenesCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if len(sceneEntities) == 0 {
		return fmt.Errorf("at least one entity is required (use -e flag)")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Generate a unique ID based on timestamp
	sceneID := strconv.FormatInt(time.Now().UnixMilli(), 10)

	// Capture current states of specified entities
	printInfo("Capturing entity states...")
	entities := make(map[string]map[string]interface{})

	for _, entityID := range sceneEntities {
		state, err := client.GetState(entityID)
		if err != nil {
			return fmt.Errorf("failed to get state for %s: %w", entityID, err)
		}

		// Build entity state for scene
		entityState := make(map[string]interface{})
		entityState["state"] = state.State

		// Include relevant attributes
		for k, v := range state.Attributes {
			// Skip non-state attributes
			if k == "friendly_name" || k == "icon" || k == "entity_id" ||
				k == "supported_features" || k == "device_class" {
				continue
			}
			entityState[k] = v
		}

		entities[entityID] = entityState
	}

	config := &api.SceneConfig{
		ID:       sceneID,
		Name:     name,
		Entities: entities,
		Icon:     sceneIcon,
	}

	printInfo("Creating scene '%s'...", name)
	if err := client.CreateScene(sceneID, config); err != nil {
		return fmt.Errorf("failed to create scene: %w", err)
	}

	fmt.Printf("Scene created: %s (ID: %s)\n", name, sceneID)
	fmt.Printf("Entity ID will be: scene.%s\n", slugify(name))
	fmt.Println("\nNote: You may need to reload scenes or restart Home Assistant for the new scene to appear.")

	return nil
}

func runScenesDelete(cmd *cobra.Command, args []string) error {
	sceneID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Deleting scene %s...", sceneID)
	if err := client.DeleteScene(sceneID); err != nil {
		return fmt.Errorf("failed to delete scene: %w", err)
	}

	fmt.Printf("Scene deleted: %s\n", sceneID)
	fmt.Println("\nNote: You may need to reload scenes or restart Home Assistant for the change to take effect.")

	return nil
}

func runScenesAddEntity(cmd *cobra.Command, args []string) error {
	sceneID := args[0]
	entityID := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Get existing scene config
	printInfo("Fetching scene configuration...")
	config, err := client.GetSceneConfig(sceneID)
	if err != nil {
		return fmt.Errorf("failed to get scene: %w", err)
	}

	// Check if entity already exists
	if _, exists := config.Entities[entityID]; exists {
		return fmt.Errorf("entity %s already exists in scene", entityID)
	}

	// Get current state of the entity
	printInfo("Capturing entity state...")
	state, err := client.GetState(entityID)
	if err != nil {
		return fmt.Errorf("failed to get entity state: %w", err)
	}

	// Build entity state
	entityState := make(map[string]interface{})
	entityState["state"] = state.State

	for k, v := range state.Attributes {
		if k == "friendly_name" || k == "icon" || k == "entity_id" ||
			k == "supported_features" || k == "device_class" {
			continue
		}
		entityState[k] = v
	}

	config.Entities[entityID] = entityState

	// Update scene
	printInfo("Updating scene...")
	if err := client.UpdateScene(sceneID, config); err != nil {
		return fmt.Errorf("failed to update scene: %w", err)
	}

	fmt.Printf("Added %s to scene %s\n", entityID, config.Name)

	return nil
}

func runScenesRemoveEntity(cmd *cobra.Command, args []string) error {
	sceneID := args[0]
	entityID := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Get existing scene config
	printInfo("Fetching scene configuration...")
	config, err := client.GetSceneConfig(sceneID)
	if err != nil {
		return fmt.Errorf("failed to get scene: %w", err)
	}

	// Check if entity exists
	if _, exists := config.Entities[entityID]; !exists {
		return fmt.Errorf("entity %s not found in scene", entityID)
	}

	delete(config.Entities, entityID)

	// Update scene
	printInfo("Updating scene...")
	if err := client.UpdateScene(sceneID, config); err != nil {
		return fmt.Errorf("failed to update scene: %w", err)
	}

	fmt.Printf("Removed %s from scene %s\n", entityID, config.Name)

	return nil
}

// slugify converts a name to a slug suitable for entity IDs.
func slugify(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces and special chars with underscores
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, slug)
	// Remove consecutive underscores
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	// Trim underscores
	slug = strings.Trim(slug, "_")
	return slug
}
