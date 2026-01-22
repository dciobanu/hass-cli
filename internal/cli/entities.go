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

var (
	entityDomain string
	entityArea   string
	entityDevice string
)

func init() {
	rootCmd.AddCommand(entitiesCmd)
	entitiesCmd.AddCommand(entitiesInspectCmd)

	entitiesCmd.Flags().StringVarP(&entityDomain, "domain", "d", "", "Filter by domain (e.g., light, switch, sensor)")
	entitiesCmd.Flags().StringVarP(&entityArea, "area", "a", "", "Filter by area name")
	entitiesCmd.Flags().StringVarP(&entityDevice, "device", "D", "", "Filter by device ID (prefix match supported)")
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
