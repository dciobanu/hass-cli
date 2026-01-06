package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var areasCmd = &cobra.Command{
	Use:   "areas",
	Short: "List all areas",
	Long: `List all areas defined in Home Assistant.

Displays area information including name, number of devices, and entities.

Examples:
  hass-cli areas              # List all areas
  hass-cli areas --json       # Output as JSON`,
	RunE: runAreas,
}

var areasInspectCmd = &cobra.Command{
	Use:   "inspect <area_id>",
	Short: "Show detailed information about an area",
	Long: `Show the complete area information including all devices and entities.

The area ID can be found by running 'hass-cli areas'.

Examples:
  hass-cli areas inspect living_room
  hass-cli areas inspect kitchen`,
	Args: cobra.ExactArgs(1),
	RunE: runAreasInspect,
}

func init() {
	rootCmd.AddCommand(areasCmd)
	areasCmd.AddCommand(areasInspectCmd)
}

// AreaWithCounts combines area info with device and entity counts.
type AreaWithCounts struct {
	AreaID      string   `json:"area_id"`
	Name        string   `json:"name"`
	FloorID     *string  `json:"floor_id"`
	Icon        *string  `json:"icon"`
	Aliases     []string `json:"aliases"`
	DeviceCount int      `json:"device_count"`
	EntityCount int      `json:"entity_count"`
}

// AreaDetail includes full area info with devices and entities.
type AreaDetail struct {
	AreaID   string           `json:"area_id"`
	Name     string           `json:"name"`
	FloorID  *string          `json:"floor_id"`
	Icon     *string          `json:"icon"`
	Aliases  []string         `json:"aliases"`
	Devices  []DeviceSummary  `json:"devices"`
	Entities []EntitySummary  `json:"entities"`
}

// DeviceSummary is a brief device representation.
type DeviceSummary struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
}

// EntitySummary is a brief entity representation.
type EntitySummary struct {
	EntityID string  `json:"entity_id"`
	Name     *string `json:"name"`
	Platform string  `json:"platform"`
}

func runAreas(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	printInfo("Connecting to Home Assistant...")
	client, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	printInfo("Fetching areas...")
	areas, err := client.GetAreas()
	if err != nil {
		return fmt.Errorf("failed to get areas: %w", err)
	}

	// Get devices and entities for counts
	devices, err := client.GetDevices()
	if err != nil {
		printInfo("Warning: could not fetch devices: %v", err)
		devices = []websocket.Device{}
	}

	entities, err := client.GetEntities()
	if err != nil {
		printInfo("Warning: could not fetch entities: %v", err)
		entities = []websocket.Entity{}
	}

	// Build device area map
	deviceAreaMap := make(map[string]string)
	for _, device := range devices {
		if device.AreaID != nil {
			deviceAreaMap[device.ID] = *device.AreaID
		}
	}

	// Count devices and entities per area
	deviceCounts := make(map[string]int)
	entityCounts := make(map[string]int)

	for _, device := range devices {
		if device.AreaID != nil {
			deviceCounts[*device.AreaID]++
		}
	}

	for _, entity := range entities {
		areaID := entity.AreaID
		// Inherit area from device if not set
		if areaID == nil && entity.DeviceID != nil {
			if deviceArea, ok := deviceAreaMap[*entity.DeviceID]; ok {
				areaID = &deviceArea
			}
		}
		if areaID != nil {
			entityCounts[*areaID]++
		}
	}

	// Build result
	var result []AreaWithCounts
	for _, area := range areas {
		result = append(result, AreaWithCounts{
			AreaID:      area.AreaID,
			Name:        area.Name,
			FloorID:     area.FloorID,
			Icon:        area.Icon,
			Aliases:     area.Aliases,
			DeviceCount: deviceCounts[area.AreaID],
			EntityCount: entityCounts[area.AreaID],
		})
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	if jsonOutput {
		return outputJSON(result)
	}

	return outputAreasTable(result)
}

func outputAreasTable(areas []AreaWithCounts) error {
	if len(areas) == 0 {
		fmt.Println("No areas found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "AREA ID\tNAME\tDEVICES\tENTITIES")
	fmt.Fprintln(w, "-------\t----\t-------\t--------")

	for _, a := range areas {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\n",
			a.AreaID,
			a.Name,
			a.DeviceCount,
			a.EntityCount,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d areas\n", len(areas))

	return nil
}

func runAreasInspect(cmd *cobra.Command, args []string) error {
	areaID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	printInfo("Connecting to Home Assistant...")
	client, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Get areas
	areas, err := client.GetAreas()
	if err != nil {
		return fmt.Errorf("failed to get areas: %w", err)
	}

	// Find the area
	var targetArea *websocket.Area
	for i := range areas {
		if areas[i].AreaID == areaID || strings.EqualFold(areas[i].Name, areaID) {
			targetArea = &areas[i]
			break
		}
	}

	if targetArea == nil {
		return fmt.Errorf("area not found: %s", areaID)
	}

	// Get devices and entities
	devices, err := client.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	entities, err := client.GetEntities()
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}

	// Build device area map
	deviceAreaMap := make(map[string]string)
	for _, device := range devices {
		if device.AreaID != nil {
			deviceAreaMap[device.ID] = *device.AreaID
		}
	}

	// Filter devices in this area
	var areaDevices []DeviceSummary
	for _, device := range devices {
		if device.AreaID != nil && *device.AreaID == targetArea.AreaID {
			areaDevices = append(areaDevices, DeviceSummary{
				ID:           device.ID,
				Name:         device.DisplayName(),
				Manufacturer: device.Manufacturer,
				Model:        device.Model,
			})
		}
	}

	// Filter entities in this area (direct or inherited from device)
	var areaEntities []EntitySummary
	for _, entity := range entities {
		entityAreaID := entity.AreaID
		if entityAreaID == nil && entity.DeviceID != nil {
			if deviceArea, ok := deviceAreaMap[*entity.DeviceID]; ok {
				entityAreaID = &deviceArea
			}
		}
		if entityAreaID != nil && *entityAreaID == targetArea.AreaID {
			areaEntities = append(areaEntities, EntitySummary{
				EntityID: entity.EntityID,
				Name:     entity.Name,
				Platform: entity.Platform,
			})
		}
	}

	// Sort
	sort.Slice(areaDevices, func(i, j int) bool {
		return areaDevices[i].Name < areaDevices[j].Name
	})
	sort.Slice(areaEntities, func(i, j int) bool {
		return areaEntities[i].EntityID < areaEntities[j].EntityID
	})

	detail := AreaDetail{
		AreaID:   targetArea.AreaID,
		Name:     targetArea.Name,
		FloorID:  targetArea.FloorID,
		Icon:     targetArea.Icon,
		Aliases:  targetArea.Aliases,
		Devices:  areaDevices,
		Entities: areaEntities,
	}

	return outputJSON(detail)
}
