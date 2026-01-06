package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [entity_id]...",
	Short: "Watch entity state changes in real-time",
	Long: `Watch for entity state changes via WebSocket.

If no entity IDs are specified, watches all state changes.
Press Ctrl+C to stop watching.

Examples:
  hass-cli watch                           # Watch all state changes
  hass-cli watch light.living_room         # Watch specific entity
  hass-cli watch light.* sensor.*          # Watch multiple patterns
  hass-cli watch --json                    # Output as JSON`,
	RunE: runWatch,
}

func init() {
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
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

	printInfo("Subscribing to state changes...")
	_, err = client.SubscribeEvents("state_changed")
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Build entity filter
	var patterns []string
	for _, arg := range args {
		patterns = append(patterns, strings.ToLower(arg))
	}

	fmt.Println("Watching for state changes... (press Ctrl+C to stop)")
	if len(patterns) > 0 {
		fmt.Printf("Filtering: %s\n", strings.Join(patterns, ", "))
	}
	fmt.Println()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Event loop
	eventChan := make(chan *websocket.EventMessage)
	errChan := make(chan error)

	go func() {
		for {
			event, err := client.ReadEvent()
			if err != nil {
				errChan <- err
				return
			}
			eventChan <- event
		}
	}()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopped watching")
			return nil

		case err := <-errChan:
			return fmt.Errorf("connection error: %w", err)

		case event := <-eventChan:
			if event.Event.EventType != "state_changed" {
				continue
			}

			entityID := event.Event.Data.EntityID

			// Apply filter
			if len(patterns) > 0 && !matchesPatterns(entityID, patterns) {
				continue
			}

			if jsonOutput {
				outputJSON(event.Event)
				continue
			}

			// Human-readable output
			newState := event.Event.Data.NewState
			oldState := event.Event.Data.OldState

			oldValue := "unavailable"
			if oldState != nil {
				oldValue = oldState.State
			}

			newValue := "unavailable"
			if newState != nil {
				newValue = newState.State
			}

			timestamp := formatEventTime(event.Event.TimeFired)
			fmt.Printf("[%s] %s: %s -> %s\n", timestamp, entityID, oldValue, newValue)
		}
	}
}

// matchesPatterns checks if an entity ID matches any of the patterns.
// Supports wildcards (*) for prefix matching.
func matchesPatterns(entityID string, patterns []string) bool {
	entityLower := strings.ToLower(entityID)

	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(entityLower, prefix) {
				return true
			}
		} else if entityLower == pattern {
			return true
		}
	}

	return false
}

// formatEventTime formats an event timestamp.
func formatEventTime(timestamp string) string {
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		// Try alternate format
		t, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return timestamp
		}
	}
	return t.Local().Format("15:04:05")
}
