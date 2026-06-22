package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

// ObjRef identifies a configuration object that references a target id.
type ObjRef struct {
	Kind     string `json:"kind"`      // scene | dashboard | script | automation
	ID       string `json:"id"`        // entity_id (scene/script/automation) or url_path (dashboard)
	ConfigID string `json:"config_id"` // numeric/object id used to fetch & edit the config
	Name     string `json:"name"`
}

// jsonContainsValue reports whether v, serialized to JSON, contains target as
// a quoted string value. Quoting avoids partial matches (e.g. light.a vs
// light.ab) — entity_ids and device_ids always appear as JSON strings.
func jsonContainsValue(v interface{}, target string) bool {
	data, err := json.Marshal(v)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), `"`+target+`"`)
}

func attrID(attrs map[string]interface{}) string {
	switch id := attrs["id"].(type) {
	case string:
		return id
	case float64:
		return fmt.Sprintf("%.0f", id)
	}
	return ""
}

func attrName(attrs map[string]interface{}) string {
	if n, ok := attrs["friendly_name"].(string); ok {
		return n
	}
	return ""
}

// findReferences returns every scene, dashboard, script, and automation whose
// configuration references target (an entity_id or device_id).
func findReferences(apiClient *api.Client, wsClient *websocket.Client, target string) ([]ObjRef, error) {
	var refs []ObjRef

	states, err := apiClient.GetStates()
	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}
	for _, s := range states {
		domain := s.EntityID
		if i := strings.IndexByte(s.EntityID, '.'); i >= 0 {
			domain = s.EntityID[:i]
		}
		switch domain {
		case "scene":
			id := attrID(s.Attributes)
			if id == "" {
				// YAML scene with no editable config; fall back to the
				// entity_id list exposed in its attributes.
				if jsonContainsValue(s.Attributes["entity_id"], target) {
					refs = append(refs, ObjRef{"scene", s.EntityID, "", attrName(s.Attributes)})
				}
				continue
			}
			cfg, err := apiClient.GetSceneConfig(id)
			if err == nil && jsonContainsValue(cfg, target) {
				refs = append(refs, ObjRef{"scene", s.EntityID, id, attrName(s.Attributes)})
			}
		case "script":
			obj := strings.TrimPrefix(s.EntityID, "script.")
			cfg, err := apiClient.GetScriptConfig(obj)
			if err == nil && jsonContainsValue(cfg, target) {
				refs = append(refs, ObjRef{"script", s.EntityID, obj, attrName(s.Attributes)})
			}
		case "automation":
			id := attrID(s.Attributes)
			if id == "" {
				continue
			}
			cfg, err := apiClient.GetAutomationConfig(id)
			if err == nil && jsonContainsValue(cfg, target) {
				refs = append(refs, ObjRef{"automation", s.EntityID, id, attrName(s.Attributes)})
			}
		}
	}

	dashboards, err := wsClient.GetDashboards()
	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}
	for _, d := range dashboards {
		cfg, err := wsClient.GetDashboardConfig(d.URLPath)
		if err == nil && jsonContainsValue(cfg, target) {
			refs = append(refs, ObjRef{"dashboard", d.URLPath, d.URLPath, d.Title})
		}
	}

	return refs, nil
}

var entitiesReferencesCmd = &cobra.Command{
	Use:   "references <entity_id|device_id>",
	Short: "Find scenes/dashboards/scripts/automations that reference an entity or device",
	Long: `Scan every scene, dashboard, script, and automation for references to a
given entity_id or device_id. Useful for impact analysis before renaming or
deleting an entity or device.

Examples:
  hass-cli entities references light.living_room_ceiling_2_0_v2
  hass-cli entities references 70113ea4072c5ebf360be6a796d3cd6b   # a device id`,
	Args: cobra.ExactArgs(1),
	RunE: runEntitiesReferences,
}

func runEntitiesReferences(cmd *cobra.Command, args []string) error {
	target := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	apiClient := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	wsClient, err := websocket.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer wsClient.Close()

	// Note whether the target still exists in the registry.
	exists := "unknown"
	if entities, err := wsClient.GetEntities(); err == nil {
		exists = "missing"
		for _, e := range entities {
			if e.EntityID == target {
				exists = "exists (entity)"
				break
			}
		}
		if exists == "missing" {
			if devices, err := wsClient.GetDevices(); err == nil {
				for _, d := range devices {
					if d.ID == target {
						exists = "exists (device)"
						break
					}
				}
			}
		}
	}

	printInfo("Scanning scenes, dashboards, scripts, automations...")
	refs, err := findReferences(apiClient, wsClient, target)
	if err != nil {
		return err
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"target":     target,
			"registry":   exists,
			"references": refs,
		})
	}

	fmt.Printf("Target: %s  (registry: %s)\n\n", target, exists)
	if len(refs) == 0 {
		fmt.Println("No references found.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tID\tNAME")
	fmt.Fprintln(w, "----\t--\t----")
	for _, r := range refs {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Kind, r.ID, r.Name)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d references\n", len(refs))
	return nil
}

func init() {
	entitiesCmd.AddCommand(entitiesReferencesCmd)
}
