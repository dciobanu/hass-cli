package cli

import (
	"fmt"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/dorinclisu/hass-cli/internal/websocket"
	"github.com/spf13/cobra"
)

var (
	migrateScenesOnly     bool
	migrateDashboardsOnly bool
	migrateDryRun         bool
)

var migrateEntityCmd = &cobra.Command{
	Use:   "migrate-entity <old_entity_id> <new_entity_id>",
	Short: "Repoint every scene and dashboard from one entity to another",
	Long: `Replace an entity_id with another across all scenes and dashboards in one
step. Scenes keep their stored per-entity state (the new entity inherits the
old one's look); dashboards have every reference swapped.

This is the bulk version of 'scenes replace-entity' + 'dashboards
replace-entity' — handy when a device is swapped for new hardware.

Scripts and automations target by area/device and usually need no change; run
'hass-cli entities references <old>' to confirm nothing else points at it.

Examples:
  hass-cli migrate-entity light.wiz_rgbw_tunable_979130 light.living_room_ceiling_1_4_v2
  hass-cli migrate-entity light.old light.new --dry-run
  hass-cli migrate-entity light.old light.new --dashboards`,
	Args: cobra.ExactArgs(2),
	RunE: runMigrateEntity,
}

func init() {
	rootCmd.AddCommand(migrateEntityCmd)
	migrateEntityCmd.Flags().BoolVar(&migrateScenesOnly, "scenes", false, "Only migrate scenes")
	migrateEntityCmd.Flags().BoolVar(&migrateDashboardsOnly, "dashboards", false, "Only migrate dashboards")
	migrateEntityCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Show what would change without saving")
}

func runMigrateEntity(cmd *cobra.Command, args []string) error {
	oldID := args[0]
	newID := args[1]
	if oldID == newID {
		return fmt.Errorf("old and new entity IDs are identical")
	}

	doScenes := !migrateDashboardsOnly
	doDashboards := !migrateScenesOnly

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

	printInfo("Scanning for references to %s...", oldID)
	refs, err := findReferences(apiClient, wsClient, oldID)
	if err != nil {
		return err
	}

	var scenes, dashboards, other []ObjRef
	for _, r := range refs {
		switch r.Kind {
		case "scene":
			scenes = append(scenes, r)
		case "dashboard":
			dashboards = append(dashboards, r)
		default:
			other = append(other, r)
		}
	}

	if len(scenes) == 0 && len(dashboards) == 0 {
		fmt.Printf("No scenes or dashboards reference %s — nothing to migrate.\n", oldID)
		if len(other) > 0 {
			fmt.Println("\nReferenced elsewhere (handle manually):")
			for _, r := range other {
				fmt.Printf("  %s: %s (%s)\n", r.Kind, r.ID, r.Name)
			}
		}
		return nil
	}

	prefix := ""
	if migrateDryRun {
		prefix = "[dry-run] "
	}

	migratedScenes := 0
	if doScenes {
		for _, s := range scenes {
			if s.ConfigID == "" {
				fmt.Printf("  %sSKIP scene %s (YAML scene, not editable via API)\n", prefix, s.ID)
				continue
			}
			if migrateDryRun {
				fmt.Printf("  %swould migrate scene %s (%s)\n", prefix, s.ID, s.Name)
				migratedScenes++
				continue
			}
			sceneCfg, err := apiClient.GetSceneConfig(s.ConfigID)
			if err != nil {
				fmt.Printf("  ERROR scene %s: %v\n", s.ID, err)
				continue
			}
			if err := replaceSceneEntityInConfig(sceneCfg, oldID, newID); err != nil {
				fmt.Printf("  ERROR scene %s: %v\n", s.ID, err)
				continue
			}
			if err := apiClient.UpdateScene(s.ConfigID, sceneCfg); err != nil {
				fmt.Printf("  ERROR scene %s: %v\n", s.ID, err)
				continue
			}
			fmt.Printf("  migrated scene %s (%s)\n", s.ID, s.Name)
			migratedScenes++
		}
	}

	migratedDashboards := 0
	if doDashboards {
		for _, d := range dashboards {
			if migrateDryRun {
				fmt.Printf("  %swould migrate dashboard %s (%s)\n", prefix, d.ID, d.Name)
				migratedDashboards++
				continue
			}
			n, err := replaceDashboardEntity(wsClient, d.ID, oldID, newID)
			if err != nil {
				fmt.Printf("  ERROR dashboard %s: %v\n", d.ID, err)
				continue
			}
			fmt.Printf("  migrated dashboard %s (%s, %d occurrence(s))\n", d.ID, d.Name, n)
			migratedDashboards++
		}
	}

	fmt.Printf("\n%s%d scene(s), %d dashboard(s) %s %s -> %s\n",
		prefix, migratedScenes, migratedDashboards,
		map[bool]string{true: "would change", false: "migrated"}[migrateDryRun], oldID, newID)

	if len(other) > 0 {
		fmt.Println("\nAlso referenced elsewhere (not migrated — handle manually):")
		for _, r := range other {
			fmt.Printf("  %s: %s (%s)\n", r.Kind, r.ID, r.Name)
		}
	}
	return nil
}
