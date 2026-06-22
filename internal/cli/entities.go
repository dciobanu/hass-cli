package cli

import (
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

var entitiesCmd = &cobra.Command{
	Use:   "entities",
	Short: "List all entities",
	Long: `List all entities registered in Home Assistant.

Displays entity information including ID, state, and area.

Examples:
  hass-cli entities              # List all entities
  hass-cli entities -d light     # Filter by domain
  hass-cli entities -a kitchen   # Filter by area
  hass-cli entities -D <device>  # Filter by device ID (prefix match)
  hass-cli entities --json       # Output as JSON`,
	RunE: runEntities,
}

var entitiesInspectCmd = &cobra.Command{
	Use:   "inspect <entity_id>",
	Short: "Show detailed information about an entity",
	Long: `Show the complete entity state and attributes as returned by the API.

Examples:
  hass-cli entities inspect light.living_room
  hass-cli entities inspect sensor.temperature`,
	Args: cobra.ExactArgs(1),
	RunE: runEntitiesInspect,
}

var entitiesRenameCmd = &cobra.Command{
	Use:   "rename <entity_id> [new_name]",
	Short: "Rename an entity",
	Long: `Rename an entity in the Home Assistant entity registry.

Set the friendly name, the entity_id (--id), or both.

Examples:
  hass-cli entities rename light.old_bulb "Spare - 1"
  hass-cli entities rename sensor.temp "Kitchen Temperature"
  hass-cli entities rename light.sengled_e11_n1ea --id light.living_room_ceiling_2_1_v2
  hass-cli entities rename light.sengled_e11_n1ea "Living Room - Ceiling - 2.1 (v2)" --id light.living_room_ceiling_2_1_v2`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runEntitiesRename,
}

var entitiesSetAreaCmd = &cobra.Command{
	Use:   "set-area <entity_id> <area_id>",
	Short: "Assign an entity to an area",
	Long: `Assign an entity to a specific area in Home Assistant.

Use an empty string or "none" to remove the area assignment.

Examples:
  hass-cli entities set-area scene.living_room_cozy living_room
  hass-cli entities set-area light.kitchen kitchen
  hass-cli entities set-area sensor.temp none    # Remove area assignment`,
	Args: cobra.ExactArgs(2),
	RunE: runEntitiesSetArea,
}

var entitiesReslugCmd = &cobra.Command{
	Use:   "reslug <device_id>",
	Short: "Align a device's entity_ids to its (renamed) device name",
	Long: `Rewrite the entity_ids of a device's entities so their object_id base
matches the device name. Renaming a device only changes friendly names; the
entity_ids keep their original slug. This fixes that in one shot.

The old base is auto-detected from the device's primary light entity (or
override with --from). The new base defaults to a slug of the device name
(override with --to). Entities whose object_id does not start with the old
base (e.g. generic signal sensors) are left untouched.

Examples:
  hass-cli entities reslug 70113ea4                                  # -> slug of device name
  hass-cli entities reslug bd8ae90d --to living_room_ceiling_1_4_v2
  hass-cli entities reslug bd8ae90d --from living_room_ceiling_1_4 --to living_room_ceiling_1_4_v2
  hass-cli entities reslug bd8ae90d --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runEntitiesReslug,
}

var (
	entityDomain string
	entityArea   string
	entityDevice string
	entityNewID  string
	reslugFrom   string
	reslugTo     string
	reslugDryRun bool
)

func init() {
	rootCmd.AddCommand(entitiesCmd)
	entitiesCmd.AddCommand(entitiesInspectCmd)
	entitiesCmd.AddCommand(entitiesRenameCmd)
	entitiesCmd.AddCommand(entitiesSetAreaCmd)
	entitiesCmd.AddCommand(entitiesReslugCmd)

	entitiesCmd.Flags().StringVarP(&entityDomain, "domain", "d", "", "Filter by domain (e.g., light, switch, sensor)")
	entitiesCmd.Flags().StringVarP(&entityArea, "area", "a", "", "Filter by area name")
	entitiesCmd.Flags().StringVarP(&entityDevice, "device", "D", "", "Filter by device ID (prefix match supported)")

	entitiesRenameCmd.Flags().StringVar(&entityNewID, "id", "", "New entity_id (changes the entity_id, not just the display name)")

	entitiesReslugCmd.Flags().StringVar(&reslugFrom, "from", "", "Old object_id base to replace (default: auto-detect from the device's light entity)")
	entitiesReslugCmd.Flags().StringVar(&reslugTo, "to", "", "New object_id base (default: slug of the device name)")
	entitiesReslugCmd.Flags().BoolVar(&reslugDryRun, "dry-run", false, "Show what would change without renaming")
}

// reslugEntityID returns the new entity_id obtained by replacing a leading
// oldBase in the object_id with newBase. The second return is false when the
// object_id does not start with oldBase (so the entity should be left alone).
func reslugEntityID(entityID, oldBase, newBase string) (string, bool) {
	dot := strings.IndexByte(entityID, '.')
	if dot < 0 {
		return entityID, false
	}
	domain, object := entityID[:dot], entityID[dot+1:]
	if object != oldBase && !strings.HasPrefix(object, oldBase+"_") {
		return entityID, false
	}
	return domain + "." + newBase + object[len(oldBase):], true
}

// EntityWithState combines entity registry info with current state.
type EntityWithState struct {
	EntityID     string                 `json:"entity_id"`
	State        string                 `json:"state"`
	AreaID       *string                `json:"area_id"`
	AreaName     string                 `json:"area_name,omitempty"`
	DeviceID     *string                `json:"device_id"`
	Platform     string                 `json:"platform"`
	Name         *string                `json:"name"`
	OriginalName *string                `json:"original_name"`
	DisabledBy   *string                `json:"disabled_by"`
	HiddenBy     *string                `json:"hidden_by"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	LastChanged  string                 `json:"last_changed,omitempty"`
}

func runEntities(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Get entity registry via WebSocket
	printInfo("Connecting to Home Assistant...")
	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	printInfo("Fetching entities...")
	entities, err := wsClient.GetEntities()
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}

	// Get areas for name resolution
	areas, err := wsClient.GetAreas()
	if err != nil {
		printInfo("Warning: could not fetch areas: %v", err)
		areas = []websocket.Area{}
	}

	// Get devices for area resolution (entities may inherit area from device)
	devices, err := wsClient.GetDevices()
	if err != nil {
		printInfo("Warning: could not fetch devices: %v", err)
		devices = []websocket.Device{}
	}

	// Build lookup maps
	areaMap := make(map[string]string)
	for _, area := range areas {
		areaMap[area.AreaID] = area.Name
	}

	deviceAreaMap := make(map[string]string)
	for _, device := range devices {
		if device.AreaID != nil {
			deviceAreaMap[device.ID] = *device.AreaID
		}
	}

	// Get current states via REST API
	restClient := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	states, err := restClient.GetStates()
	if err != nil {
		printInfo("Warning: could not fetch states: %v", err)
		states = []api.State{}
	}

	stateMap := make(map[string]api.State)
	for _, state := range states {
		stateMap[state.EntityID] = state
	}

	// Combine entity registry with states
	var combined []EntityWithState
	for _, entity := range entities {
		// Get area (from entity or inherited from device)
		areaID := entity.AreaID
		if areaID == nil && entity.DeviceID != nil {
			if deviceArea, ok := deviceAreaMap[*entity.DeviceID]; ok {
				areaID = &deviceArea
			}
		}

		var areaName string
		if areaID != nil {
			areaName = areaMap[*areaID]
		}

		state := stateMap[entity.EntityID]

		ews := EntityWithState{
			EntityID:     entity.EntityID,
			State:        state.State,
			AreaID:       areaID,
			AreaName:     areaName,
			DeviceID:     entity.DeviceID,
			Platform:     entity.Platform,
			Name:         entity.Name,
			OriginalName: entity.GetOriginalName(),
			DisabledBy:   entity.DisabledBy,
			HiddenBy:     entity.HiddenBy,
			LastChanged:  state.LastChanged,
		}

		// Apply filters
		if entityDomain != "" {
			parts := strings.SplitN(entity.EntityID, ".", 2)
			if len(parts) < 2 || !strings.EqualFold(parts[0], entityDomain) {
				continue
			}
		}

		if entityArea != "" {
			if areaName == "" {
				continue
			}
			if !strings.Contains(strings.ToLower(areaName), strings.ToLower(entityArea)) {
				continue
			}
		}

		if entityDevice != "" {
			if entity.DeviceID == nil {
				continue
			}
			// Support prefix match
			if *entity.DeviceID != entityDevice && !strings.HasPrefix(*entity.DeviceID, entityDevice) {
				continue
			}
		}

		combined = append(combined, ews)
	}

	// Sort by entity_id
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].EntityID < combined[j].EntityID
	})

	if jsonOutput {
		return outputJSON(combined)
	}

	return outputEntitiesTable(combined)
}

func outputEntitiesTable(entities []EntityWithState) error {
	if len(entities) == 0 {
		fmt.Println("No entities found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENTITY ID\tSTATE\tNAME\tAREA")
	fmt.Fprintln(w, "---------\t-----\t----\t----")

	for _, e := range entities {
		name := ""
		if e.Name != nil && *e.Name != "" {
			name = *e.Name
		} else if e.OriginalName != nil && *e.OriginalName != "" {
			name = *e.OriginalName
		}
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		state := e.State
		if len(state) > 15 {
			state = state[:12] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			e.EntityID,
			state,
			name,
			e.AreaName,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d entities\n", len(entities))

	return nil
}

func runEntitiesInspect(cmd *cobra.Command, args []string) error {
	entityID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching entity state...")
	state, err := client.GetState(entityID)
	if err != nil {
		return fmt.Errorf("failed to get entity: %w", err)
	}

	return outputJSON(state)
}

func runEntitiesRename(cmd *cobra.Command, args []string) error {
	entityID := args[0]

	updates := make(map[string]interface{})
	var newName string
	if len(args) == 2 {
		newName = args[1]
		updates["name"] = newName
	}
	if entityNewID != "" {
		updates["new_entity_id"] = entityNewID
	}

	if len(updates) == 0 {
		return fmt.Errorf("nothing to change: provide a new name and/or --id")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	_, err = wsClient.UpdateEntity(entityID, updates)
	if err != nil {
		return fmt.Errorf("failed to rename entity: %w", err)
	}

	switch {
	case newName != "" && entityNewID != "":
		fmt.Printf("Renamed %s to: %s (new entity_id: %s)\n", entityID, newName, entityNewID)
	case entityNewID != "":
		fmt.Printf("Changed entity_id %s to: %s\n", entityID, entityNewID)
	default:
		fmt.Printf("Renamed %s to: %s\n", entityID, newName)
	}
	return nil
}

func runEntitiesReslug(cmd *cobra.Command, args []string) error {
	deviceID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	devices, err := wsClient.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}
	var device *websocket.Device
	for i := range devices {
		if devices[i].ID == deviceID || strings.HasPrefix(devices[i].ID, deviceID) {
			device = &devices[i]
			break
		}
	}
	if device == nil {
		return fmt.Errorf("no device found with ID: %s", deviceID)
	}

	entities, err := wsClient.GetEntities()
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}
	var deviceEntities []websocket.Entity
	for _, e := range entities {
		if e.DeviceID != nil && *e.DeviceID == device.ID {
			deviceEntities = append(deviceEntities, e)
		}
	}
	if len(deviceEntities) == 0 {
		return fmt.Errorf("device %s has no entities", device.ID)
	}

	// Determine the old base: explicit --from, else the object_id of the
	// device's primary light entity.
	oldBase := reslugFrom
	if oldBase == "" {
		for _, e := range deviceEntities {
			if strings.HasPrefix(e.EntityID, "light.") {
				oldBase = strings.TrimPrefix(e.EntityID, "light.")
				break
			}
		}
		if oldBase == "" {
			return fmt.Errorf("could not auto-detect the old base (no light entity); pass --from")
		}
	}

	newBase := reslugTo
	if newBase == "" {
		newBase = slugify(device.DisplayName())
	}
	if newBase == oldBase {
		fmt.Printf("Nothing to do: base already %q\n", newBase)
		return nil
	}

	renamed, skipped := 0, 0
	for _, e := range deviceEntities {
		newID, ok := reslugEntityID(e.EntityID, oldBase, newBase)
		if !ok {
			skipped++
			continue
		}
		if reslugDryRun {
			fmt.Printf("  [dry-run] %s -> %s\n", e.EntityID, newID)
			renamed++
			continue
		}
		if _, err := wsClient.UpdateEntity(e.EntityID, map[string]interface{}{"new_entity_id": newID}); err != nil {
			fmt.Printf("  ERROR %s: %v\n", e.EntityID, err)
			continue
		}
		fmt.Printf("  %s -> %s\n", e.EntityID, newID)
		renamed++
	}

	verb := "renamed"
	if reslugDryRun {
		verb = "would rename"
	}
	fmt.Printf("\n%s %d entit(y/ies) (%s -> %s); %d left unchanged\n", verb, renamed, oldBase, newBase, skipped)
	return nil
}

func runEntitiesSetArea(cmd *cobra.Command, args []string) error {
	entityID := args[0]
	areaID := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	// Handle "none" or empty string to clear area assignment
	updates := make(map[string]interface{})
	if areaID == "" || strings.ToLower(areaID) == "none" {
		updates["area_id"] = nil
	} else {
		// Validate area exists
		areas, err := wsClient.GetAreas()
		if err != nil {
			return fmt.Errorf("failed to get areas: %w", err)
		}

		var foundArea *websocket.Area
		for _, area := range areas {
			if area.AreaID == areaID || strings.EqualFold(area.Name, areaID) {
				foundArea = &area
				break
			}
		}

		if foundArea == nil {
			return fmt.Errorf("area not found: %s", areaID)
		}

		updates["area_id"] = foundArea.AreaID
		areaID = foundArea.AreaID // Use the actual ID
	}

	_, err = wsClient.UpdateEntity(entityID, updates)
	if err != nil {
		return fmt.Errorf("failed to set area: %w", err)
	}

	if areaID == "" || strings.ToLower(args[1]) == "none" {
		fmt.Printf("Removed area assignment from %s\n", entityID)
	} else {
		fmt.Printf("Assigned %s to area: %s\n", entityID, areaID)
	}

	return nil
}
