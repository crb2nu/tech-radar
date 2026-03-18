// Package main provides the CLI for Tech Radar.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	techradar "gitlab.flexinfer.ai/services/tech-radar"
	"gitlab.flexinfer.ai/services/tech-radar/internal/radar"
	"gitlab.flexinfer.ai/services/tech-radar/internal/scanner"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
	"gopkg.in/yaml.v3"
)

var (
	version = "dev"
	dataDir string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "radar",
		Short:   "Tech Radar - Personal technology decision tracker",
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Ensure dataDir is absolute if possible, but for now just use it
		},
	}

	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", ".tech-radar", "Directory for radar data and config")

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
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return err
			}

			// ... (default radar setup)
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

			if err := os.WriteFile(filepath.Join(dataDir, "radar.yaml"), radarData, 0644); err != nil {
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
						".mcp.json",
						"mcp.json",
						"claude_desktop_config.json",
						"cline_mcp_settings.json",
					},
					Mappings: map[string]types.EntryHint{
						"react":                       {Name: "React", Quadrant: types.QuadrantFrameworks},
						"typescript":                  {Name: "TypeScript", Quadrant: types.QuadrantLanguages},
						"openai":                      {Name: "OpenAI API", Quadrant: types.QuadrantPlatforms},
						"@anthropic-ai/sdk":           {Name: "Claude API", Quadrant: types.QuadrantPlatforms},
						"langchain":                   {Name: "LangChain", Quadrant: types.QuadrantFrameworks},
						"langgraph":                   {Name: "LangGraph", Quadrant: types.QuadrantFrameworks},
						"ollama":                      {Name: "Ollama", Quadrant: types.QuadrantPlatforms},
						"vllm":                        {Name: "vLLM", Quadrant: types.QuadrantPlatforms},
						"promptfoo":                   {Name: "Promptfoo", Quadrant: types.QuadrantTools},
						"litellm":                     {Name: "LiteLLM", Quadrant: types.QuadrantPlatforms},
						"github.com/mark3labs/mcp-go": {Name: "mcp-go", Quadrant: types.QuadrantTools},
						"@modelcontextprotocol/sdk":   {Name: "Model Context Protocol", Quadrant: types.QuadrantTools},
						"@modelcontextprotocol/server-filesystem": {Name: "Filesystem MCP Server", Quadrant: types.QuadrantTools},
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

			if err := os.WriteFile(filepath.Join(dataDir, "config.yaml"), configData, 0644); err != nil {
				return err
			}

			fmt.Printf("Initialized tech radar in %s/\n", dataDir)
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
		tags     string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add or update a technology entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			radar, err := loadRadar()
			if err != nil {
				return fmt.Errorf("radar not initialized, run 'radar init' first")
			}

			// Ring order for movement calculation (center to edge)
			ringOrder := map[types.Ring]int{
				types.RingAdopt:  0,
				types.RingTrial:  1,
				types.RingAssess: 2,
				types.RingHold:   3,
			}

			tagList := []string{}
			if tags != "" {
				for _, t := range strings.Split(tags, ",") {
					tagList = append(tagList, strings.TrimSpace(t))
				}
			}

			found := false
			newRing := types.Ring(ring)
			for i, entry := range radar.Entries {
				if strings.EqualFold(entry.Name, name) {
					// Update existing entry
					oldRing := entry.Ring
					radar.Entries[i].Ring = newRing
					if quadrant != "" {
						radar.Entries[i].Quadrant = types.Quadrant(quadrant)
					}
					if reason != "" {
						radar.Entries[i].Description = reason
					}
					if len(tagList) > 0 {
						radar.Entries[i].Tags = tagList
					}

					// Calculate movement
					if oldRing != newRing {
						if ringOrder[newRing] < ringOrder[oldRing] {
							radar.Entries[i].Moved = 1 // Moved in
						} else {
							radar.Entries[i].Moved = -1 // Moved out
						}
						radar.Entries[i].History = append(radar.Entries[i].History, types.HistoryEntry{
							Date:   time.Now(),
							Ring:   newRing,
							Reason: reason,
						})
					} else {
						radar.Entries[i].Moved = 0
					}
					found = true
					fmt.Printf("Updated %s: %s -> %s\n", entry.Name, oldRing, newRing)
					break
				}
			}

			if !found {
				// Add new entry
				entry := types.Entry{
					Name:        name,
					Quadrant:    types.Quadrant(quadrant),
					Ring:        newRing,
					Description: reason,
					Tags:        tagList,
					Moved:       0,
					History: []types.HistoryEntry{
						{
							Date:   time.Now(),
							Ring:   newRing,
							Reason: reason,
						},
					},
				}
				radar.Entries = append(radar.Entries, entry)
				fmt.Printf("Added %s to %s ring\n", name, ring)
			}

			radar.Meta.Updated = time.Now()
			return saveRadar(radar)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Technology name")
	cmd.Flags().StringVar(&quadrant, "quadrant", "tools", "Quadrant (languages, frameworks, tools, platforms)")
	cmd.Flags().StringVar(&ring, "ring", "assess", "Ring (adopt, trial, assess, hold)")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for adding/updating")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags")
	cmd.MarkFlagRequired("name")

	return cmd
}

func scanCmd() *cobra.Command {
	var (
		autoAdd bool
		publish bool
		apiURL  string
		apiKey  string
		project string
	)

	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Scan projects for technologies",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config if available
			var config types.Config
			// Default config
			config.Scan.Files = []string{
				"package.json",
				"go.mod",
				"requirements.txt",
				"pyproject.toml",
				"Cargo.toml",
				".mcp.json",
				"mcp.json",
				"claude_desktop_config.json",
				"cline_mcp_settings.json",
			}

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

				if publish {
					if apiURL == "" {
						return fmt.Errorf("--publish requires --api-url")
					}
					// Send to API
					scanResult := types.ScanResult{
						ProjectID:    project,
						Path:         path,
						Technologies: result.Technologies,
					}
					if err := publishResult(apiURL, apiKey, scanResult); err != nil {
						fmt.Printf("Error publishing results: %v\n", err)
					} else {
						fmt.Println("Successfully published results to central server")
					}
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

	cmd.Flags().BoolVar(&autoAdd, "auto-add", false, "Automatically add/update discovered technologies")
	cmd.Flags().BoolVar(&publish, "publish", false, "Publish results to a central radar server")
	cmd.Flags().StringVar(&apiURL, "api-url", "", "Radar API URL (e.g. http://radar.internal/api/v1/ingest)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API Key for authentication")
	cmd.Flags().StringVar(&project, "project", "", "Project name/ID for this scan")

	return cmd
}

func publishResult(apiURL, apiKey string, result types.ScanResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	// Check if response is JSON to avoid false successes from Cloudflare/proxies
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("server returned non-JSON response (%s), likely intercepted by a proxy or login page", contentType)
	}

	return nil
}

func addToRadar(techs []types.DiscoveredTech) error {
	radar, err := loadRadar()
	if err != nil {
		return fmt.Errorf("radar not initialized, run 'radar init' first")
	}

	addedCount := 0

	for _, tech := range techs {
		found := false
		for i, entry := range radar.Entries {
			if strings.EqualFold(entry.Name, tech.Name) {
				found = true
				if tech.EvidencePath != "" && radar.Entries[i].Link == "" {
					radar.Entries[i].Link = tech.EvidencePath
					if !containsTag(radar.Entries[i].Tags, "adr") {
						radar.Entries[i].Tags = append(radar.Entries[i].Tags, "adr")
					}
					if tech.EvidenceTitle != "" && radar.Entries[i].Description == "" {
						radar.Entries[i].Description = fmt.Sprintf("Decision captured in %s", tech.EvidenceTitle)
					}
					addedCount++
				}
				break
			}
		}

		if found {
			continue
		}

		// Add new entry
		quadrant := tech.Quadrant
		if quadrant == "" {
			quadrant = types.QuadrantTools
		}

		entry := types.Entry{
			Name:        tech.Name,
			Quadrant:    quadrant,
			Ring:        types.RingAssess,
			Description: discoveryDescription(tech),
			Moved:       0,
			Link:        tech.EvidencePath,
			Tags:        autoTags(tech),
			History: []types.HistoryEntry{
				{
					Date:   time.Now(),
					Ring:   types.RingAssess,
					Reason: discoveryDescription(tech),
				},
			},
		}

		radar.Entries = append(radar.Entries, entry)
		addedCount++
	}

	if addedCount > 0 {
		radar.Meta.Updated = time.Now()
		if err := saveRadar(radar); err != nil {
			return err
		}
		fmt.Printf("Added %d new entries to radar\n", addedCount)
	} else {
		fmt.Println("No new technologies discovered")
	}

	return nil
}

func discoveryDescription(tech types.DiscoveredTech) string {
	if tech.EvidenceTitle != "" {
		return fmt.Sprintf("Decision captured in %s", tech.EvidenceTitle)
	}
	if tech.EvidencePath != "" {
		return fmt.Sprintf("Decision captured in %s", tech.EvidencePath)
	}
	return fmt.Sprintf("Auto-discovered from %s", tech.Source)
}

func autoTags(tech types.DiscoveredTech) []string {
	if tech.EvidencePath == "" {
		return nil
	}
	return []string{"adr"}
}

func containsTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, target) {
			return true
		}
	}
	return false
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
	var standalone bool

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
			// Inject data
			script := fmt.Sprintf("<script>window.RADAR_DATA = %s;</script>", string(jsonData))
			html = strings.Replace(html, "<head>", "<head>"+script, 1)

			if standalone {
				// Inline CSS
				reCSS := regexp.MustCompile(`<link rel="stylesheet" [^>]*href="\./radar-assets/([^"]+)"[^>]*>`)
				html = reCSS.ReplaceAllStringFunc(html, func(match string) string {
					submatch := reCSS.FindStringSubmatch(match)
					if len(submatch) > 1 {
						content, err := techradar.UI.ReadFile("web/dist/radar-assets/" + submatch[1])
						if err == nil {
							return "<style>" + string(content) + "</style>"
						}
					}
					return match
				})

				// Inline JS
				reJS := regexp.MustCompile(`<script [^>]*src="\./radar-assets/([^"]+)"[^>]*></script>`)
				html = reJS.ReplaceAllStringFunc(html, func(match string) string {
					submatch := reJS.FindStringSubmatch(match)
					if len(submatch) > 1 {
						content, err := techradar.UI.ReadFile("web/dist/radar-assets/" + submatch[1])
						if err == nil {
							// For standalone, we might need to strip 'type="module"' if we want it to work
							// in all contexts, but for now just inlining is fine.
							return "<script>" + string(content) + "</script>"
						}
					}
					return match
				})
			} else {
				// Copy assets as before
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
			}

			if err := os.WriteFile(output, []byte(html), 0644); err != nil {
				return err
			}

			if standalone {
				fmt.Printf("Rendered standalone radar to %s\n", output)
			} else {
				fmt.Printf("Rendered radar to %s (assets in radar-assets/)\n", output)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "radar.html", "Output file")
	cmd.Flags().BoolVarP(&standalone, "standalone", "s", false, "Inline all assets into a single HTML file")

	return cmd
}

func serveCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve interactive radar",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := radar.NewServer(port, dataDir, techradar.UI)
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
