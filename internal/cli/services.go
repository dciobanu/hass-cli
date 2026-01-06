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

var servicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List available services",
	Long: `List all available services in Home Assistant.

Services are organized by domain (e.g., light, switch, scene).

Examples:
  hass-cli services              # List all services
  hass-cli services -d light     # Filter by domain
  hass-cli services --json       # Output as JSON`,
	RunE: runServices,
}

var servicesInspectCmd = &cobra.Command{
	Use:   "inspect <domain.service>",
	Short: "Show detailed information about a service",
	Long: `Show detailed information about a service including its fields.

Examples:
  hass-cli services inspect light.turn_on
  hass-cli services inspect scene.turn_on`,
	Args: cobra.ExactArgs(1),
	RunE: runServicesInspect,
}

var serviceDomain string

func init() {
	rootCmd.AddCommand(servicesCmd)
	servicesCmd.AddCommand(servicesInspectCmd)

	servicesCmd.Flags().StringVarP(&serviceDomain, "domain", "d", "", "Filter by domain (e.g., light, switch, scene)")
}

// ServiceListItem represents a service for listing.
type ServiceListItem struct {
	Domain      string `json:"domain"`
	Service     string `json:"service"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func runServices(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching services...")
	services, err := client.GetServices()
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	// Build flat list
	var items []ServiceListItem
	for domain, svcMap := range services {
		// Filter by domain if specified
		if serviceDomain != "" && !strings.EqualFold(domain, serviceDomain) {
			continue
		}

		for svcName, svcInfo := range svcMap {
			items = append(items, ServiceListItem{
				Domain:      domain,
				Service:     svcName,
				Name:        svcInfo.Name,
				Description: svcInfo.Description,
			})
		}
	}

	// Sort by domain.service
	sort.Slice(items, func(i, j int) bool {
		if items[i].Domain != items[j].Domain {
			return items[i].Domain < items[j].Domain
		}
		return items[i].Service < items[j].Service
	})

	if jsonOutput {
		return outputJSON(items)
	}

	return outputServicesTable(items)
}

func outputServicesTable(services []ServiceListItem) error {
	if len(services) == 0 {
		fmt.Println("No services found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tNAME\tDESCRIPTION")
	fmt.Fprintln(w, "-------\t----\t-----------")

	for _, s := range services {
		name := s.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}

		desc := s.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		fmt.Fprintf(w, "%s.%s\t%s\t%s\n",
			s.Domain,
			s.Service,
			name,
			desc,
		)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d services\n", len(services))

	return nil
}

// ServiceDetail contains detailed service info.
type ServiceDetail struct {
	Domain      string                      `json:"domain"`
	Service     string                      `json:"service"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Fields      map[string]api.ServiceField `json:"fields"`
	Target      *api.ServiceTarget          `json:"target"`
}

func runServicesInspect(cmd *cobra.Command, args []string) error {
	fullService := args[0]

	parts := strings.SplitN(fullService, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid service format: %s (expected domain.service)", fullService)
	}
	domain := parts[0]
	service := parts[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	client := api.NewClient(cfg.Server.URL, cfg.Server.Token, time.Duration(timeout)*time.Second)

	printInfo("Fetching service details...")
	services, err := client.GetServices()
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	domainServices, ok := services[domain]
	if !ok {
		return fmt.Errorf("domain not found: %s", domain)
	}

	svcInfo, ok := domainServices[service]
	if !ok {
		return fmt.Errorf("service not found: %s.%s", domain, service)
	}

	detail := ServiceDetail{
		Domain:      domain,
		Service:     service,
		Name:        svcInfo.Name,
		Description: svcInfo.Description,
		Fields:      svcInfo.Fields,
		Target:      svcInfo.Target,
	}

	if jsonOutput {
		return outputJSON(detail)
	}

	// Human-readable output
	fmt.Printf("Service:       %s.%s\n", domain, service)
	fmt.Printf("Name:          %s\n", svcInfo.Name)
	fmt.Printf("Description:   %s\n", svcInfo.Description)

	if svcInfo.Target != nil {
		fmt.Println("\nTarget:")
		if len(svcInfo.Target.Entity) > 0 {
			fmt.Println("  - Entities")
		}
		if len(svcInfo.Target.Device) > 0 {
			fmt.Println("  - Devices")
		}
		if len(svcInfo.Target.Area) > 0 {
			fmt.Println("  - Areas")
		}
	}

	if len(svcInfo.Fields) > 0 {
		fmt.Println("\nFields:")
		for name, field := range svcInfo.Fields {
			required := ""
			if field.Required {
				required = " (required)"
			}
			fmt.Printf("  %s%s\n", name, required)
			if field.Description != "" {
				fmt.Printf("    %s\n", field.Description)
			}
			if field.Example != nil {
				fmt.Printf("    Example: %v\n", field.Example)
			}
		}
	}

	return nil
}
