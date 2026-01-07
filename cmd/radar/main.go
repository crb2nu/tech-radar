// Package main provides the CLI for Tech Radar.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
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
			for _, path := range args {
				fmt.Printf("Scanning %s...\n", path)
				// TODO: Implement scanning
				fmt.Println("  (scanning not yet implemented)")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&autoAdd, "auto-add", false, "Automatically add discovered technologies")

	return cmd
}

func renderCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "render",
		Short: "Generate radar visualization",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement SVG/HTML rendering
			fmt.Printf("Rendering to %s...\n", output)
			fmt.Println("(rendering not yet implemented)")
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
			fmt.Printf("Serving radar on http://localhost:%d\n", port)
			fmt.Println("(server not yet implemented)")
			return nil
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
