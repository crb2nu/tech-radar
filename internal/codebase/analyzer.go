// Package codebase synthesizes project scan data into visualization-ready
// structures: dependency graphs, tech-stack matrices, and pattern detection.
package codebase

import (
	"math"
	"sort"
	"strings"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

// patternRule defines a heuristic for detecting an architectural pattern.
type patternRule struct {
	Name   string
	Icon   string
	Signal string
	Match  func(techs []string, sources []string) bool
}

var defaultPatternRules = []patternRule{
	{
		Name:   "GitOps",
		Icon:   "🔄",
		Signal: "Flux, ArgoCD, or GitOps tooling detected",
		Match: func(techs, sources []string) bool {
			keywords := []string{"flux", "argocd", "argo-cd", "gitops", "kustomize"}
			return anyContains(techs, keywords)
		},
	},
	{
		Name:   "MCP-enabled",
		Icon:   "🤖",
		Signal: "Model Context Protocol config files or SDK detected",
		Match: func(techs, sources []string) bool {
			mcpSources := []string{".mcp.json", "mcp.json", "claude_desktop_config.json", "cline_mcp_settings.json"}
			if anyExact(sources, mcpSources) {
				return true
			}
			keywords := []string{"model context protocol", "mcp-go", "modelcontextprotocol"}
			return anyContains(techs, keywords)
		},
	},
	{
		Name:   "AI/ML pipeline",
		Icon:   "🧠",
		Signal: "AI/ML frameworks or inference platforms detected",
		Match: func(techs, _ []string) bool {
			keywords := []string{"openai", "anthropic", "langchain", "langgraph", "ollama", "vllm", "litellm", "llamaindex", "llama-index", "pytorch", "tensorflow", "transformers", "huggingface"}
			return anyContains(techs, keywords)
		},
	},
	{
		Name:   "Containerized",
		Icon:   "📦",
		Signal: "Docker or container runtime tooling detected",
		Match: func(techs, _ []string) bool {
			keywords := []string{"docker", "podman", "buildkit", "kaniko", "containerd"}
			return anyContains(techs, keywords)
		},
	},
	{
		Name:   "Monorepo",
		Icon:   "🏗️",
		Signal: "Multiple language manifests in same project",
		Match: func(_, sources []string) bool {
			langManifests := map[string]bool{}
			mapping := map[string]string{
				"go.mod":           "go",
				"package.json":     "js",
				"pyproject.toml":   "python",
				"requirements.txt": "python",
				"Cargo.toml":       "rust",
			}
			for _, s := range sources {
				if lang, ok := mapping[s]; ok {
					langManifests[lang] = true
				}
			}
			return len(langManifests) >= 2
		},
	},
	{
		Name:   "Full-stack",
		Icon:   "🌐",
		Signal: "Both frontend and backend frameworks detected",
		Match: func(techs, _ []string) bool {
			hasFrontend := anyContains(techs, []string{"react", "svelte", "vue", "angular", "next", "nuxt", "astro"})
			hasBackend := anyContains(techs, []string{"express", "fastapi", "django", "flask", "gin", "echo", "fiber", "cobra"})
			return hasFrontend && hasBackend
		},
	},
}

// Analyze creates a CodebaseAnalysis from radar entries and project data.
func Analyze(radar *types.Radar, projects []types.Project) *types.CodebaseAnalysis {
	if radar == nil {
		radar = &types.Radar{}
	}

	entryByName := make(map[string]types.Entry, len(radar.Entries))
	for _, e := range radar.Entries {
		entryByName[strings.ToLower(e.Name)] = e
	}

	// 1. Build nodes: every unique technology across all projects.
	techUsage := map[string]int{}
	techSources := map[string][]string{}
	for _, p := range projects {
		seen := map[string]bool{}
		for _, t := range p.Technologies {
			key := strings.ToLower(t.Name)
			if !seen[key] {
				techUsage[key]++
				seen[key] = true
			}
			techSources[key] = appendUnique(techSources[key], t.Source)
		}
	}

	nodes := make([]types.TechNode, 0, len(techUsage))
	trackedCount := 0
	for techKey, usage := range techUsage {
		entry, tracked := entryByName[techKey]
		node := types.TechNode{
			Name:      bestName(techKey, entry, tracked),
			Quadrant:  string(entry.Quadrant),
			Ring:      string(entry.Ring),
			Usage:     usage,
			IsTracked: tracked,
		}
		if !tracked {
			node.Quadrant = guessQuadrant(techKey, techSources[techKey])
		}
		nodes = append(nodes, node)
		if tracked {
			trackedCount++
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Usage != nodes[j].Usage {
			return nodes[i].Usage > nodes[j].Usage
		}
		return nodes[i].Name < nodes[j].Name
	})

	// 2. Build edges: technologies that appear together in the same project.
	edges := buildEdges(projects, entryByName)

	// 3. Build per-project stacks.
	stacks := buildStacks(projects, entryByName)

	// 4. Detect patterns.
	patterns := detectPatterns(projects, entryByName)

	// 5. Build file trees.
	fileTrees := buildFileTrees(projects)

	// 6. Summary.
	trackedPercent := 0.0
	if len(techUsage) > 0 {
		trackedPercent = math.Round(float64(trackedCount)/float64(len(techUsage))*1000) / 10
	}

	return &types.CodebaseAnalysis{
		Nodes:    nodes,
		Edges:    edges,
		Stacks:   stacks,
		Patterns: patterns,
		FileTree: fileTrees,
		Summary: types.CodebaseSummary{
			TotalProjects:     len(projects),
			TotalTechnologies: len(nodes),
			TotalEdges:        len(edges),
			TotalPatterns:     len(patterns),
			TrackedPercent:    trackedPercent,
		},
	}
}

func buildEdges(projects []types.Project, entryByName map[string]types.Entry) []types.DependencyEdge {
	type edgeKey struct{ a, b string }
	edgeMap := map[edgeKey][]string{}

	for _, p := range projects {
		techNames := make([]string, 0, len(p.Technologies))
		seen := map[string]bool{}
		for _, t := range p.Technologies {
			key := strings.ToLower(t.Name)
			if seen[key] {
				continue
			}
			seen[key] = true
			// Use canonical name from radar if available.
			name := t.Name
			if e, ok := entryByName[key]; ok {
				name = e.Name
			}
			techNames = append(techNames, name)
		}

		// Generate pairwise edges (only for techs used ≥2 times across projects to limit noise).
		for i := 0; i < len(techNames); i++ {
			for j := i + 1; j < len(techNames); j++ {
				a, b := techNames[i], techNames[j]
				if a > b {
					a, b = b, a
				}
				k := edgeKey{a, b}
				edgeMap[k] = appendUnique(edgeMap[k], p.Name)
			}
		}
	}

	edges := make([]types.DependencyEdge, 0, len(edgeMap))
	for k, projs := range edgeMap {
		if len(projs) < 1 {
			continue
		}
		edges = append(edges, types.DependencyEdge{
			Source:   k.a,
			Target:   k.b,
			Weight:   len(projs),
			Projects: projs,
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Weight > edges[j].Weight
	})

	// Limit to top 200 edges to keep the graph manageable.
	if len(edges) > 200 {
		edges = edges[:200]
	}

	return edges
}

func buildStacks(projects []types.Project, entryByName map[string]types.Entry) []types.ProjectStack {
	stacks := make([]types.ProjectStack, 0, len(projects))
	for _, p := range projects {
		categories := map[string][]string{}
		for _, t := range p.Technologies {
			key := strings.ToLower(t.Name)
			name := t.Name
			quadrant := guessQuadrant(key, []string{t.Source})
			if e, ok := entryByName[key]; ok {
				name = e.Name
				quadrant = string(e.Quadrant)
			}
			if quadrant == "" {
				quadrant = "tools"
			}
			categories[quadrant] = appendUnique(categories[quadrant], name)
		}

		stack := types.ProjectStack{
			ProjectID:   p.ID,
			ProjectName: p.Name,
			LastScan:    p.LastScan,
			Categories:  categories,
			TechCount:   len(p.Technologies),
		}

		stacks = append(stacks, stack)
	}

	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].TechCount > stacks[j].TechCount
	})

	return stacks
}

func detectPatterns(projects []types.Project, entryByName map[string]types.Entry) []types.PatternTag {
	var patterns []types.PatternTag

	for _, rule := range defaultPatternRules {
		var matchedProjects []string

		for _, p := range projects {
			techs := make([]string, 0, len(p.Technologies))
			sources := make([]string, 0, len(p.Technologies))
			for _, t := range p.Technologies {
				techs = append(techs, strings.ToLower(t.Name))
				sources = append(sources, strings.ToLower(t.Source))
			}
			// Also check radar entry names for this project.
			for _, t := range p.Technologies {
				key := strings.ToLower(t.Name)
				if e, ok := entryByName[key]; ok {
					techs = append(techs, strings.ToLower(e.Name))
				}
			}

			if rule.Match(techs, sources) {
				matchedProjects = append(matchedProjects, p.Name)
			}
		}

		if len(matchedProjects) > 0 {
			patterns = append(patterns, types.PatternTag{
				Pattern:  rule.Name,
				Projects: matchedProjects,
				Signal:   rule.Signal,
				Icon:     rule.Icon,
			})
		}
	}

	sort.Slice(patterns, func(i, j int) bool {
		return len(patterns[i].Projects) > len(patterns[j].Projects)
	})

	return patterns
}

func buildFileTrees(projects []types.Project) []types.ProjectTree {
	trees := make([]types.ProjectTree, 0, len(projects))
	for _, p := range projects {
		fileMap := map[string][]string{}
		for _, t := range p.Technologies {
			src := t.Source
			if src == "" {
				src = "unknown"
			}
			fileMap[src] = appendUnique(fileMap[src], t.Name)
		}

		files := make([]types.FileNode, 0, len(fileMap))
		for path, techs := range fileMap {
			sort.Strings(techs)
			files = append(files, types.FileNode{
				Path:         path,
				Technologies: techs,
			})
		}
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		trees = append(trees, types.ProjectTree{
			ProjectID:   p.ID,
			ProjectName: p.Name,
			Files:       files,
		})
	}

	sort.Slice(trees, func(i, j int) bool {
		return trees[i].ProjectName < trees[j].ProjectName
	})

	return trees
}

// guessQuadrant tries to categorize untracked techs based on source file type.
func guessQuadrant(name string, sources []string) string {
	for _, s := range sources {
		switch strings.ToLower(s) {
		case "go.mod", "package.json", "pyproject.toml", "requirements.txt", "cargo.toml":
			return "frameworks"
		case ".mcp.json", "mcp.json", "claude_desktop_config.json", "cline_mcp_settings.json":
			return "tools"
		case "adr":
			return "techniques"
		}
	}
	return "tools"
}

func bestName(key string, entry types.Entry, tracked bool) string {
	if tracked {
		return entry.Name
	}
	// Capitalize first letter as a fallback.
	if len(key) == 0 {
		return key
	}
	return strings.ToUpper(key[:1]) + key[1:]
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return slice
		}
	}
	return append(slice, val)
}

func anyContains(haystack []string, needles []string) bool {
	for _, h := range haystack {
		for _, n := range needles {
			if strings.Contains(h, n) {
				return true
			}
		}
	}
	return false
}

func anyExact(haystack []string, needles []string) bool {
	for _, h := range haystack {
		for _, n := range needles {
			if strings.EqualFold(h, n) {
				return true
			}
		}
	}
	return false
}
