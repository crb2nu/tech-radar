// Package main provides the CLI for Tech Radar.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	techradar "gitlab.flexinfer.ai/services/tech-radar"
	"gitlab.flexinfer.ai/services/tech-radar/internal/radar"
	"gitlab.flexinfer.ai/services/tech-radar/internal/scanner"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
	"gopkg.in/yaml.v3"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "radar",
		Short:   "Tech Radar - Personal technology decision tracker",
		Version: version,
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(scanCmd())
	rootCmd.AddCommand(renderCmd())
	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(listCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize tech radar configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			radarDir := ".tech-radar"

			if err := os.MkdirAll(radarDir, 0755); err != nil {
				return err
			}

			// Create default radar
			radar := types.Radar{
				Meta: types.RadarMeta{
					Name:    "Personal Tech Radar",
					Updated: time.Now(),
					Version: "1.0.0",
				},
				Entries: []types.Entry{
					{
						Name:        "Go",
						Quadrant:    types.QuadrantLanguages,
						Ring:        types.RingAdopt,
						Description: "Primary backend language",
					},
					{
						Name:        "Python",
						Quadrant:    types.QuadrantLanguages,
						Ring:        types.RingAdopt,
						Description: "Data processing and scripting",
					},
					{
						Name:        "TypeScript",
						Quadrant:    types.QuadrantLanguages,
						Ring:        types.RingAdopt,
						Description: "Frontend and Node.js development",
					},
				},
			}

			radarData, err := yaml.Marshal(radar)
			if err != nil {
				return err
			}

			if err := os.WriteFile(radarDir+"/radar.yaml", radarData, 0644); err != nil {
				return err
			}

			// Create default config
			config := types.Config{
				Scan: types.ScanConfig{
					Files: []string{
						"package.json",
						"go.mod",
						"requirements.txt",
						"pyproject.toml",
						"Cargo.toml",
					},
					Mappings: map[string]types.EntryHint{
						"react":      {Name: "React", Quadrant: types.QuadrantFrameworks},
						"typescript": {Name: "TypeScript", Quadrant: types.QuadrantLanguages},
					},
				},
				Visualization: types.VisualizationConfig{
					Colors: map[string]string{
						"adopt":  "#93c47d",
						"trial":  "#93d2c2",
						"assess": "#fbdb84",
						"hold":   "#efafa9",
					},
					Title:    "Tech Radar",
					ShowDots: true,
				},
			}

			configData, err := yaml.Marshal(config)
			if err != nil {
				return err
			}

			if err := os.WriteFile(radarDir+"/config.yaml", configData, 0644); err != nil {
				return err
			}

			fmt.Printf("Initialized tech radar in %s/\n", radarDir)
			return nil
		},
	}
}

func addCmd() *cobra.Command {
	var (
		name     string
		quadrant string
		ring     string
		reason   string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a technology entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load existing radar
			data, err := os.ReadFile(".tech-radar/radar.yaml")
			if err != nil {
				return fmt.Errorf("radar not initialized, run 'radar init' first")
			}

			var radar types.Radar
			if err := yaml.Unmarshal(data, &radar); err != nil {
				return err
			}

			// Add new entry
			entry := types.Entry{
				Name:        name,
				Quadrant:    types.Quadrant(quadrant),
				Ring:        types.Ring(ring),
				Description: reason,
				Moved:       1,
				History: []types.HistoryEntry{
					{
						Date:   time.Now(),
						Ring:   types.Ring(ring),
						Reason: reason,
					},
				},
			}

			radar.Entries = append(radar.Entries, entry)
			radar.Meta.Updated = time.Now()

			// Save
			newData, err := yaml.Marshal(radar)
			if err != nil {
				return err
			}

			if err := os.WriteFile(".tech-radar/radar.yaml", newData, 0644); err != nil {
				return err
			}

			fmt.Printf("Added %s to %s ring\n", name, ring)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Technology name")
	cmd.Flags().StringVar(&quadrant, "quadrant", "tools", "Quadrant (languages, frameworks, tools, platforms, techniques)")
	cmd.Flags().StringVar(&ring, "ring", "assess", "Ring (adopt, trial, assess, hold)")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for adding")
	cmd.MarkFlagRequired("name")

	return cmd
}

func scanCmd() *cobra.Command {
	var autoAdd bool

	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Scan projects for technologies",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config if available
			var config types.Config
			// Default config
			config.Scan.Files = []string{"package.json", "go.mod", "requirements.txt", "pyproject.toml", "Cargo.toml"}

			if data, err := os.ReadFile(".tech-radar/config.yaml"); err == nil {
				yaml.Unmarshal(data, &config)
			}

			s := scanner.New(config.Scan)

			for _, path := range args {
				fmt.Printf("Scanning %s...\n", path)
				result, err := s.Scan(path)
				if err != nil {
					fmt.Printf("Error scanning %s: %v\n", path, err)
					continue
				}

				fmt.Printf("Found %d technologies:\n", len(result.Technologies))
				for _, tech := range result.Technologies {
					fmt.Printf("  - %s (%s)\n", tech.Name, tech.Source)
				}

				if autoAdd && len(result.Technologies) > 0 {
					if err := addToRadar(result.Technologies); err != nil {
						fmt.Printf("Error adding to radar: %v\n", err)
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoAdd, "auto-add", false, "Automatically add discovered technologies")

	return cmd
}

func addToRadar(techs []types.DiscoveredTech) error {
	radar, err := loadRadar()
	if err != nil {
		return fmt.Errorf("radar not initialized, run 'radar init' first")
	}

	existing := make(map[string]bool)
	for _, entry := range radar.Entries {
		existing[strings.ToLower(entry.Name)] = true
	}

	addedCount := 0
	for _, tech := range techs {
		if existing[strings.ToLower(tech.Name)] {
			continue
		}

		quadrant := tech.Quadrant
		if quadrant == "" {
			quadrant = types.QuadrantTools
		}

		entry := types.Entry{
			Name:        tech.Name,
			Quadrant:    quadrant,
			Ring:        types.RingAssess,
			Description: fmt.Sprintf("Auto-discovered from %s", tech.Source),
			Moved:       1,
			History: []types.HistoryEntry{
				{
					Date:   time.Now(),
					Ring:   types.RingAssess,
					Reason: fmt.Sprintf("Auto-discovered from %s", tech.Source),
				},
			},
		}

		radar.Entries = append(radar.Entries, entry)
		existing[strings.ToLower(tech.Name)] = true
		addedCount++
	}

	if addedCount > 0 {
		radar.Meta.Updated = time.Now()
		if err := saveRadar(radar); err != nil {
			return err
		}
		fmt.Printf("Added %d new entries to radar\n", addedCount)
	} else {
		fmt.Println("No new entries to add")
	}

	return nil
}

func loadRadar() (*types.Radar, error) {
	data, err := os.ReadFile(".tech-radar/radar.yaml")
	if err != nil {
		return nil, err
	}

	var radar types.Radar
	if err := yaml.Unmarshal(data, &radar); err != nil {
		return nil, err
	}
	return &radar, nil
}

func saveRadar(radar *types.Radar) error {
	data, err := yaml.Marshal(radar)
	if err != nil {
		return err
	}
	return os.WriteFile(".tech-radar/radar.yaml", data, 0644)
}

func renderCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Generate radar visualization",
		RunE: func(cmd *cobra.Command, args []string) error {
			radarData, err := os.ReadFile(".tech-radar/radar.yaml")
			if err != nil {
				return fmt.Errorf("radar not initialized")
			}

			var r types.Radar
			if err := yaml.Unmarshal(radarData, &r); err != nil {
				return err
			}

			jsonData, err := json.Marshal(r)
			if err != nil {
				return err
			}

			// Read index.html from embedded assets
			indexData, err := techradar.UI.ReadFile("web/dist/index.html")
			if err != nil {
				return fmt.Errorf("failed to read index.html: %v", err)
			}

			html := string(indexData)
			// Inject data - we'll need the frontend to look for window.RADAR_DATA
			script := fmt.Sprintf("<script>window.RADAR_DATA = %s;</script>", string(jsonData))
			html = strings.Replace(html, "<head>", "<head>"+script, 1)

			// Also need to handle other assets (JS/CSS)
			// For a truly portable HTML, we might want to inline them,
			// but for now, we'll just assume they are relative.
			// However, since we're using go:embed, they are already built.

			if err := os.WriteFile(output, []byte(html), 0644); err != nil {
				return err
			}

			// Copy assets
			entries, err := techradar.UI.ReadDir("web/dist/radar-assets")
			if err == nil {
				os.MkdirAll("radar-assets", 0755)
				for _, entry := range entries {
					data, err := techradar.UI.ReadFile("web/dist/radar-assets/" + entry.Name())
					if err == nil {
						os.WriteFile("radar-assets/"+entry.Name(), data, 0644)
					}
				}
			}

			fmt.Printf("Rendered radar to %s\n", output)
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "radar.html", "Output file")

	return cmd
}

func serveCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve interactive radar",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := radar.NewServer(port, techradar.UI)
			return server.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 8088, "Server port")

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all radar entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(".tech-radar/radar.yaml")
			if err != nil {
				return fmt.Errorf("radar not initialized")
			}

			var radar types.Radar
			if err := yaml.Unmarshal(data, &radar); err != nil {
				return err
			}

			fmt.Printf("Tech Radar: %s\n\n", radar.Meta.Name)

			for _, ring := range []types.Ring{types.RingAdopt, types.RingTrial, types.RingAssess, types.RingHold} {
				fmt.Printf("=== %s ===\n", ring)
				for _, entry := range radar.Entries {
					if entry.Ring == ring {
						fmt.Printf("  %-20s [%s] %s\n", entry.Name, entry.Quadrant, entry.Description)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}
}
