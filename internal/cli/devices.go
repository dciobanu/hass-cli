package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/config"
	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List all registered devices",
	Long: `List all devices registered in Home Assistant.

Displays device information including name, manufacturer, model, and area.

Examples:
  hass-cli devices              # List all devices
  hass-cli devices --json       # Output as JSON
  hass-cli devices -m philips   # Filter by manufacturer`,
	RunE: runDevices,
}

var devicesInspectCmd = &cobra.Command{
	Use:   "inspect <device_id>",
	Short: "Show detailed information about a device",
	Long: `Show the complete device information as returned by the API.

The device ID can be found by running 'hass-cli devices'.
You can use a partial ID (prefix match) for convenience.

Examples:
  hass-cli devices inspect 4ee3f48beb2fcdeee4f8195b8f1730da
  hass-cli devices inspect 4ee3f48b    # Prefix match`,
	Args: cobra.ExactArgs(1),
	RunE: runDevicesInspect,
}

var devicesRemoveCmd = &cobra.Command{
	Use:   "remove <device_id>",
	Short: "Remove a device from the registry",
	Long: `Remove a device from the Home Assistant device registry.

This removes all config entry associations from the device. When a device
has no more config entries, it is automatically deleted by Home Assistant.

Warning: This may affect the integration that manages this device.

The device ID can be found by running 'hass-cli devices'.
You can use a partial ID (prefix match) for convenience.

Examples:
  hass-cli devices remove 4ee3f48beb2fcdeee4f8195b8f1730da
  hass-cli devices remove 4ee3f48b    # Prefix match`,
	Args: cobra.ExactArgs(1),
	RunE: runDevicesRemove,
}

var devicesDisableCmd = &cobra.Command{
	Use:   "disable <device_id>",
	Short: "Disable a device",
	Long: `Disable a device in Home Assistant.

Disabled devices and their entities will not be available in Home Assistant
until re-enabled. This is useful for temporarily disabling devices without
removing them.

The device ID can be found by running 'hass-cli devices'.
You can use a partial ID (prefix match) for convenience.

Examples:
  hass-cli devices disable 4ee3f48beb2fcdeee4f8195b8f1730da
  hass-cli devices disable 4ee3f48b    # Prefix match`,
	Args: cobra.ExactArgs(1),
	RunE: runDevicesDisable,
}

var devicesEnableCmd = &cobra.Command{
	Use:   "enable <device_id>",
	Short: "Enable a disabled device",
	Long: `Enable a previously disabled device in Home Assistant.

The device ID can be found by running 'hass-cli devices'.
You can use a partial ID (prefix match) for convenience.

Examples:
  hass-cli devices enable 4ee3f48beb2fcdeee4f8195b8f1730da
  hass-cli devices enable 4ee3f48b    # Prefix match`,
	Args: cobra.ExactArgs(1),
	RunE: runDevicesEnable,
}

var (
	deviceManufacturer string
	deviceArea         string
)

func init() {
	rootCmd.AddCommand(devicesCmd)
	devicesCmd.AddCommand(devicesInspectCmd)
	devicesCmd.AddCommand(devicesRemoveCmd)
	devicesCmd.AddCommand(devicesDisableCmd)
	devicesCmd.AddCommand(devicesEnableCmd)

	devicesCmd.Flags().StringVarP(&deviceManufacturer, "manufacturer", "m", "", "Filter by manufacturer (case-insensitive)")
	devicesCmd.Flags().StringVarP(&deviceArea, "area", "a", "", "Filter by area ID")
}

func runDevices(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Create WebSocket client
	printInfo("Connecting to Home Assistant...")
	client, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Get devices
	printInfo("Fetching devices...")
	devices, err := client.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	// Get areas for resolving area names
	areas, err := client.GetAreas()
	if err != nil {
		printInfo("Warning: could not fetch areas: %v", err)
		areas = []websocket.Area{}
	}

	// Build area lookup map
	areaMap := make(map[string]string)
	for _, area := range areas {
		areaMap[area.AreaID] = area.Name
	}

	// Filter devices
	filtered := filterDevices(devices, areaMap)

	// Sort by name
	sort.Slice(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].DisplayName()) < strings.ToLower(filtered[j].DisplayName())
	})

	// Output
	if jsonOutput {
		return outputJSON(filtered)
	}

	return outputDevicesTable(filtered, areaMap)
}

func filterDevices(devices []websocket.Device, areaMap map[string]string) []websocket.Device {
	if deviceManufacturer == "" && deviceArea == "" {
		return devices
	}

	filtered := make([]websocket.Device, 0)
	manufacturerLower := strings.ToLower(deviceManufacturer)

	for _, d := range devices {
		// Filter by manufacturer
		if deviceManufacturer != "" {
			if d.Manufacturer == nil || !strings.Contains(strings.ToLower(*d.Manufacturer), manufacturerLower) {
				continue
			}
		}

		// Filter by area
		if deviceArea != "" {
			if d.AreaID == nil || *d.AreaID != deviceArea {
				// Also check by area name
				if d.AreaID == nil {
					continue
				}
				areaName := areaMap[*d.AreaID]
				if !strings.EqualFold(areaName, deviceArea) && !strings.Contains(strings.ToLower(areaName), strings.ToLower(deviceArea)) {
					continue
				}
			}
		}

		filtered = append(filtered, d)
	}

	return filtered
}

func outputDevicesTable(devices []websocket.Device, areaMap map[string]string) error {
	if len(devices) == 0 {
		fmt.Println("No devices found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tMANUFACTURER\tMODEL\tAREA")
	fmt.Fprintln(w, "--\t----\t------------\t-----\t----")

	for _, d := range devices {
		area := ""
		if d.AreaID != nil {
			if name, ok := areaMap[*d.AreaID]; ok {
				area = name
			} else {
				area = *d.AreaID
			}
		}

		name := d.DisplayName()
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		manufacturer := d.DisplayManufacturer()
		if len(manufacturer) > 18 {
			manufacturer = manufacturer[:15] + "..."
		}

		model := d.DisplayModel()
		if len(model) > 18 {
			model = model[:15] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			d.ID,
			name,
			manufacturer,
			model,
			area,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d devices\n", len(devices))

	return nil
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func runDevicesInspect(cmd *cobra.Command, args []string) error {
	deviceID := args[0]

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Create WebSocket client
	printInfo("Connecting to Home Assistant...")
	client, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Get devices
	printInfo("Fetching devices...")
	devices, err := client.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	// Find device by ID (exact or prefix match)
	var found *websocket.Device
	var matches []websocket.Device

	for i := range devices {
		if devices[i].ID == deviceID {
			// Exact match
			found = &devices[i]
			break
		}
		if strings.HasPrefix(devices[i].ID, deviceID) {
			matches = append(matches, devices[i])
		}
	}

	// If no exact match, check prefix matches
	if found == nil {
		if len(matches) == 0 {
			return fmt.Errorf("no device found with ID: %s", deviceID)
		}
		if len(matches) > 1 {
			fmt.Fprintf(os.Stderr, "Multiple devices match '%s':\n", deviceID)
			for _, d := range matches {
				fmt.Fprintf(os.Stderr, "  %s  %s\n", d.ID, d.DisplayName())
			}
			return fmt.Errorf("please provide a more specific ID")
		}
		found = &matches[0]
	}

	// Output the device as formatted JSON
	return outputJSON(found)
}

func runDevicesRemove(cmd *cobra.Command, args []string) error {
	deviceID := args[0]

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Create WebSocket client
	printInfo("Connecting to Home Assistant...")
	client, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Get devices to resolve partial ID and show name
	printInfo("Fetching devices...")
	devices, err := client.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	// Find device by ID (exact or prefix match)
	var found *websocket.Device
	var matches []websocket.Device

	for i := range devices {
		if devices[i].ID == deviceID {
			found = &devices[i]
			break
		}
		if strings.HasPrefix(devices[i].ID, deviceID) {
			matches = append(matches, devices[i])
		}
	}

	if found == nil {
		if len(matches) == 0 {
			return fmt.Errorf("no device found with ID: %s", deviceID)
		}
		if len(matches) > 1 {
			fmt.Fprintf(os.Stderr, "Multiple devices match '%s':\n", deviceID)
			for _, d := range matches {
				fmt.Fprintf(os.Stderr, "  %s  %s\n", d.ID, d.DisplayName())
			}
			return fmt.Errorf("please provide a more specific ID")
		}
		found = &matches[0]
	}

	// Check if device has config entries
	if len(found.ConfigEntries) == 0 {
		return fmt.Errorf("device has no config entries - it may already be orphaned or managed differently")
	}

	// Remove all config entries from the device
	printInfo("Removing device %s (%s)...", found.ID, found.DisplayName())
	for _, configEntryID := range found.ConfigEntries {
		printInfo("  Removing config entry %s...", configEntryID)
		if err := client.RemoveConfigEntryFromDevice(found.ID, configEntryID); err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "does not support device removal") {
				return fmt.Errorf("integration does not support device removal via API - use the Home Assistant UI or remove the integration")
			}
			return fmt.Errorf("failed to remove config entry %s: %w", configEntryID, err)
		}
	}

	fmt.Printf("Device removed: %s (%s)\n", found.ID, found.DisplayName())
	return nil
}

func runDevicesDisable(cmd *cobra.Command, args []string) error {
	return setDeviceDisabled(args[0], true)
}

func runDevicesEnable(cmd *cobra.Command, args []string) error {
	return setDeviceDisabled(args[0], false)
}

func setDeviceDisabled(deviceID string, disable bool) error {
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

	printInfo("Fetching devices...")
	devices, err := client.GetDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	// Find device by ID (exact or prefix match)
	var found *websocket.Device
	var matches []websocket.Device

	for i := range devices {
		if devices[i].ID == deviceID {
			found = &devices[i]
			break
		}
		if strings.HasPrefix(devices[i].ID, deviceID) {
			matches = append(matches, devices[i])
		}
	}

	if found == nil {
		if len(matches) == 0 {
			return fmt.Errorf("no device found with ID: %s", deviceID)
		}
		if len(matches) > 1 {
			fmt.Fprintf(os.Stderr, "Multiple devices match '%s':\n", deviceID)
			for _, d := range matches {
				fmt.Fprintf(os.Stderr, "  %s  %s\n", d.ID, d.DisplayName())
			}
			return fmt.Errorf("please provide a more specific ID")
		}
		found = &matches[0]
	}

	var device *websocket.Device
	if disable {
		printInfo("Disabling device %s (%s)...", found.ID, found.DisplayName())
		device, err = client.DisableDevice(found.ID)
		if err != nil {
			return fmt.Errorf("failed to disable device: %w", err)
		}
		fmt.Printf("Device disabled: %s (%s)\n", device.ID, device.DisplayName())
	} else {
		printInfo("Enabling device %s (%s)...", found.ID, found.DisplayName())
		device, err = client.EnableDevice(found.ID)
		if err != nil {
			return fmt.Errorf("failed to enable device: %w", err)
		}
		fmt.Printf("Device enabled: %s (%s)\n", device.ID, device.DisplayName())
	}

	return nil
}

// loadConfig loads the configuration, respecting command-line overrides.
func loadConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	// Load from file
	if configPath != "" {
		cfg, err = config.LoadFrom(configPath)
	} else {
		cfg, err = config.Load()
	}

	// If config doesn't exist but URL and token are provided via flags, create a temporary config
	if err == config.ErrNotConfigured && serverURL != "" && token != "" {
		cfg = &config.Config{
			Server: config.ServerConfig{
				URL:   serverURL,
				Token: token,
			},
			Defaults: config.DefaultsConfig{
				Output:  "human",
				Timeout: timeout,
			},
		}
		err = nil
	}

	if err != nil {
		return nil, err
	}

	// Apply command-line overrides
	if serverURL != "" {
		cfg.Server.URL = serverURL
	}
	if token != "" {
		cfg.Server.Token = token
	}

	// Validate
	if !cfg.IsConfigured() {
		return nil, config.ErrNotConfigured
	}

	return cfg, nil
}
