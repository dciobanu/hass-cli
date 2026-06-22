package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dorinclisu/hass-cli/internal/api"
	"github.com/spf13/cobra"
)

var configEntryDomain string

var configEntriesCmd = &cobra.Command{
	Use:   "config-entries",
	Short: "List and remove integration config entries",
	Long: `List and remove Home Assistant integration config entries.

Some integrations (e.g. WiZ) register each device as its own config entry and
do not support device removal via the device API. Removing the config entry is
the way to delete such a device and all its entities.

Examples:
  hass-cli config-entries                      # List all config entries
  hass-cli config-entries -d wiz               # Filter by integration domain
  hass-cli config-entries remove <entry_id>    # Delete a config entry`,
	RunE: runConfigEntries,
}

var configEntriesRemoveCmd = &cobra.Command{
	Use:   "remove <entry_id>",
	Short: "Remove (delete) an integration config entry",
	Long: `Remove an integration config entry, deleting its devices and entities.

Warning: this is destructive. For integrations with one device per entry
(e.g. WiZ) it deletes exactly that device.

Examples:
  hass-cli config-entries remove 32ab67516cddb90bb4b79f63abc67636`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigEntriesRemove,
}

func init() {
	rootCmd.AddCommand(configEntriesCmd)
	configEntriesCmd.AddCommand(configEntriesRemoveCmd)
	configEntriesCmd.Flags().StringVarP(&configEntryDomain, "domain", "d", "", "Filter by integration domain (e.g. wiz)")
}

func runConfigEntries(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching config entries...")
	entries, err := client.ListConfigEntries()
	if err != nil {
		return fmt.Errorf("failed to list config entries: %w", err)
	}

	if configEntryDomain != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if strings.EqualFold(e.Domain, configEntryDomain) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Domain != entries[j].Domain {
			return entries[i].Domain < entries[j].Domain
		}
		return entries[i].Title < entries[j].Title
	})

	if jsonOutput {
		return outputJSON(entries)
	}
	if len(entries) == 0 {
		fmt.Println("No config entries found")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENTRY ID\tDOMAIN\tTITLE\tSTATE")
	fmt.Fprintln(w, "--------\t------\t-----\t-----")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.EntryID, e.Domain, e.Title, e.State)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d config entries\n", len(entries))
	return nil
}

func runConfigEntriesRemove(cmd *cobra.Command, args []string) error {
	entryID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	// Resolve a friendly label for the message (best effort).
	label := entryID
	if entries, err := client.ListConfigEntries(); err == nil {
		for _, e := range entries {
			if e.EntryID == entryID {
				label = fmt.Sprintf("%s (%s)", e.Title, e.Domain)
				break
			}
		}
	}

	printInfo("Removing config entry %s...", entryID)
	if err := client.RemoveConfigEntry(entryID); err != nil {
		return fmt.Errorf("failed to remove config entry: %w", err)
	}
	fmt.Printf("Config entry removed: %s\n", label)
	return nil
}
