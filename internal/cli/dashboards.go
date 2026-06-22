package cli

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var dashboardsCmd = &cobra.Command{
	Use:   "dashboards",
	Short: "List and manage Lovelace dashboards",
	Long: `List and manage Home Assistant Lovelace dashboards.

Only storage-mode dashboards can be inspected and edited. The default
"Overview" dashboard is referenced with an empty url path ("").

Examples:
  hass-cli dashboards                                          # List dashboards
  hass-cli dashboards inspect dashboard-upstairs               # Show entities used
  hass-cli dashboards inspect dashboard-upstairs --json        # Show full config
  hass-cli dashboards replace-entity dashboard-upstairs light.old light.new`,
	RunE: runDashboards,
}

var dashboardsInspectCmd = &cobra.Command{
	Use:   "inspect <url_path>",
	Short: "Show a dashboard's entities (or full config with --json)",
	Long: `Show the entities referenced by a dashboard.

By default this lists the distinct entities referenced and flags any that
no longer exist in the entity registry. Use --json for the raw config.

Examples:
  hass-cli dashboards inspect dashboard-upstairs
  hass-cli dashboards inspect dashboard-upstairs --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDashboardsInspect,
}

var dashboardsReplaceEntityCmd = &cobra.Command{
	Use:   "replace-entity <url_path> <old_entity_id> <new_entity_id>",
	Short: "Replace every reference to an entity across a dashboard",
	Long: `Replace every reference to an entity_id in a dashboard's configuration
with another entity_id. Useful when a device is swapped and its dashboard
cards point at the now-deleted entity.

Examples:
  hass-cli dashboards replace-entity dashboard-upstairs light.wiz_old light.sengled_new`,
	Args: cobra.ExactArgs(3),
	RunE: runDashboardsReplaceEntity,
}

func init() {
	rootCmd.AddCommand(dashboardsCmd)
	dashboardsCmd.AddCommand(dashboardsInspectCmd)
	dashboardsCmd.AddCommand(dashboardsReplaceEntityCmd)
}

func newWSClient() (*websocket.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
}

func runDashboards(cmd *cobra.Command, args []string) error {
	wsClient, err := newWSClient()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	printInfo("Fetching dashboards...")
	dashboards, err := wsClient.GetDashboards()
	if err != nil {
		return fmt.Errorf("failed to get dashboards: %w", err)
	}

	sort.Slice(dashboards, func(i, j int) bool {
		return dashboards[i].Title < dashboards[j].Title
	})

	if jsonOutput {
		return outputJSON(dashboards)
	}

	if len(dashboards) == 0 {
		fmt.Println("No dashboards found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "URL PATH\tTITLE\tMODE\tSIDEBAR")
	fmt.Fprintln(w, "--------\t-----\t----\t-------")
	for _, d := range dashboards {
		sidebar := "no"
		if d.ShowInSidebar {
			sidebar = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.URLPath, d.Title, d.Mode, sidebar)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d dashboards\n", len(dashboards))
	return nil
}

func runDashboardsInspect(cmd *cobra.Command, args []string) error {
	urlPath := args[0]

	wsClient, err := newWSClient()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	printInfo("Fetching dashboard config...")
	config, err := wsClient.GetDashboardConfig(urlPath)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %w", err)
	}

	if jsonOutput {
		return outputJSON(config)
	}

	// Collect distinct entity references and check existence.
	refs := collectEntityRefs(config)
	if len(refs) == 0 {
		fmt.Println("No entity references found in dashboard")
		return nil
	}

	printInfo("Fetching entity registry...")
	entities, err := wsClient.GetEntities()
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}
	existing := make(map[string]bool, len(entities))
	for _, e := range entities {
		existing[e.EntityID] = true
	}

	sorted := make([]string, 0, len(refs))
	for r := range refs {
		sorted = append(sorted, r)
	}
	sort.Strings(sorted)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENTITY\tREFS\tSTATUS")
	fmt.Fprintln(w, "------\t----\t------")
	missing := 0
	for _, r := range sorted {
		status := "ok"
		if !existing[r] {
			status = "MISSING"
			missing++
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", r, refs[r], status)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d entities (%d missing)\n", len(sorted), missing)
	return nil
}

func runDashboardsReplaceEntity(cmd *cobra.Command, args []string) error {
	urlPath := args[0]
	oldEntityID := args[1]
	newEntityID := args[2]

	if oldEntityID == newEntityID {
		return fmt.Errorf("old and new entity IDs are identical")
	}

	wsClient, err := newWSClient()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	printInfo("Fetching dashboard config...")
	config, err := wsClient.GetDashboardConfig(urlPath)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %w", err)
	}

	count := 0
	replaced := replaceEntityRefs(config, oldEntityID, newEntityID, &count)
	if count == 0 {
		return fmt.Errorf("entity %s not found in dashboard %s", oldEntityID, urlPath)
	}

	printInfo("Saving dashboard config...")
	if err := wsClient.SaveDashboardConfig(urlPath, replaced.(map[string]interface{})); err != nil {
		return fmt.Errorf("failed to save dashboard: %w", err)
	}

	fmt.Printf("Replaced %s with %s in dashboard %s (%d occurrence(s))\n", oldEntityID, newEntityID, urlPath, count)
	return nil
}

// replaceDashboardEntity swaps oldEntityID for newEntityID throughout a
// dashboard config and saves it, returning the number of occurrences replaced.
func replaceDashboardEntity(wsClient *websocket.Client, urlPath, oldEntityID, newEntityID string) (int, error) {
	config, err := wsClient.GetDashboardConfig(urlPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get dashboard: %w", err)
	}
	count := 0
	replaced := replaceEntityRefs(config, oldEntityID, newEntityID, &count)
	if count == 0 {
		return 0, nil
	}
	if err := wsClient.SaveDashboardConfig(urlPath, replaced.(map[string]interface{})); err != nil {
		return 0, fmt.Errorf("failed to save dashboard: %w", err)
	}
	return count, nil
}

// isEntityID reports whether s looks like a Home Assistant entity_id
// (domain.object_id).
func isEntityID(s string) bool {
	dot := -1
	for i, r := range s {
		switch {
		case r == '.':
			if dot != -1 {
				return false // more than one dot
			}
			dot = i
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			// allowed
		default:
			return false
		}
	}
	return dot > 0 && dot < len(s)-1
}

// collectEntityRefs walks a dashboard config and counts strings that look
// like entity IDs.
func collectEntityRefs(node interface{}) map[string]int {
	refs := make(map[string]int)
	var walk func(interface{})
	walk = func(n interface{}) {
		switch v := n.(type) {
		case map[string]interface{}:
			for _, child := range v {
				walk(child)
			}
		case []interface{}:
			for _, child := range v {
				walk(child)
			}
		case string:
			if isEntityID(v) {
				refs[v]++
			}
		}
	}
	walk(node)
	return refs
}

// replaceEntityRefs returns a copy of node with every string equal to
// oldID replaced by newID, incrementing count for each replacement.
func replaceEntityRefs(node interface{}, oldID, newID string, count *int) interface{} {
	switch v := node.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, child := range v {
			out[k] = replaceEntityRefs(child, oldID, newID, count)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, child := range v {
			out[i] = replaceEntityRefs(child, oldID, newID, count)
		}
		return out
	case string:
		if v == oldID {
			*count++
			return newID
		}
		return v
	default:
		return node
	}
}
