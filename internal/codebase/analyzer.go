// Package codebase synthesizes project scan data into visualization-ready
// structures: dependency graphs, tech-stack matrices, and pattern detection.
package codebase

import (
	"math"
	"sort"
	"strings"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

type projectFindingRule struct {
	ID           string
	PatternName  string
	Icon         string
	Signal       string
	Kind         types.FindingKind
	Severity     types.FindingSeverity
	Confidence   float64
	Title        string
	Alternatives []string
	Match        func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence
	Summary      func(project types.Project, evidence []types.FindingEvidence) string
}

var projectFindingRules = []projectFindingRule{
	{
		ID:          "gitops",
		PatternName: "GitOps",
		Icon:        "🔄",
		Signal:      "Flux, ArgoCD, or GitOps tooling detected",
		Kind:        types.FindingKindPattern,
		Severity:    types.FindingSeverityLow,
		Confidence:  0.95,
		Title:       "GitOps architecture",
		Alternatives: []string{
			"Document the approved GitOps workflow in an ADR.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectTechEvidence(project, func(tech types.DiscoveredTech) bool {
				return techKeywordMatch(tech, []string{"flux", "argocd", "argo-cd", "gitops", "kustomize"})
			})
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "Flux, ArgoCD, or Kustomize signals show a GitOps-style delivery path."
		},
	},
	{
		ID:          "mcp-enabled",
		PatternName: "MCP-enabled",
		Icon:        "🤖",
		Signal:      "Model Context Protocol config files or SDK detected",
		Kind:        types.FindingKindPattern,
		Severity:    types.FindingSeverityLow,
		Confidence:  0.93,
		Title:       "MCP-enabled workspace",
		Alternatives: []string{
			"Standardize the MCP launcher and config path for the team.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectTechEvidence(project, func(tech types.DiscoveredTech) bool {
				if sourceExactMatch(tech.Source, []string{".mcp.json", "mcp.json", "claude_desktop_config.json", "cline_mcp_settings.json"}) {
					return true
				}
				return techKeywordMatch(tech, []string{"model context protocol", "mcp-go", "modelcontextprotocol"})
			})
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "Model Context Protocol tooling is present across config or SDK files."
		},
	},
	{
		ID:          "ai-ml-pipeline",
		PatternName: "AI/ML pipeline",
		Icon:        "🧠",
		Signal:      "AI/ML frameworks or inference platforms detected",
		Kind:        types.FindingKindPattern,
		Severity:    types.FindingSeverityLow,
		Confidence:  0.90,
		Title:       "AI/ML pipeline",
		Alternatives: []string{
			"Prefer one approved model/runtime path for this workflow.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectTechEvidence(project, func(tech types.DiscoveredTech) bool {
				return techKeywordMatch(tech, []string{"openai", "anthropic", "langchain", "langgraph", "ollama", "vllm", "litellm", "llamaindex", "llama-index", "pytorch", "tensorflow", "transformers", "huggingface"})
			})
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "AI/ML frameworks or inference platforms are part of the project stack."
		},
	},
	{
		ID:          "containerized",
		PatternName: "Containerized",
		Icon:        "📦",
		Signal:      "Docker or container runtime tooling detected",
		Kind:        types.FindingKindPattern,
		Severity:    types.FindingSeverityLow,
		Confidence:  0.88,
		Title:       "Containerized runtime",
		Alternatives: []string{
			"Keep the image and build contract in one shared base.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectTechEvidence(project, func(tech types.DiscoveredTech) bool {
				return techKeywordMatch(tech, []string{"docker", "podman", "buildkit", "kaniko", "containerd"})
			})
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "Container tooling is part of the project delivery path."
		},
	},
	{
		ID:          "monorepo",
		PatternName: "Monorepo",
		Icon:        "🏗️",
		Signal:      "Multiple language manifests in same project",
		Kind:        types.FindingKindPattern,
		Severity:    types.FindingSeverityLow,
		Confidence:  0.86,
		Title:       "Monorepo shape",
		Alternatives: []string{
			"Keep shared tooling documented if this split is intentional.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectFamilyEvidence(project, 2)
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "This project spans multiple manifest ecosystems, which suggests a monorepo or polyglot service boundary."
		},
	},
	{
		ID:          "full-stack",
		PatternName: "Full-stack",
		Icon:        "🌐",
		Signal:      "Both frontend and backend frameworks detected",
		Kind:        types.FindingKindPattern,
		Severity:    types.FindingSeverityLow,
		Confidence:  0.84,
		Title:       "Full-stack delivery",
		Alternatives: []string{
			"Document the split between frontend and backend ownership.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectFullStackEvidence(project)
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "Frontend and backend frameworks co-exist in the same project."
		},
	},
	{
		ID:         "hold-usage",
		Kind:       types.FindingKindAntiPattern,
		Severity:   types.FindingSeverityHigh,
		Confidence: 0.99,
		Title:      "Hold technology still in use",
		Alternatives: []string{
			"Replace the hold tech with the approved adopt standard.",
			"Set a migration checkpoint before the usage spreads further.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectHoldEvidence(project, entryByName)
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "This project still depends on hold-ring technologies, which usually means migration work is overdue."
		},
	},
	{
		ID:         "untracked-concentration",
		Kind:       types.FindingKindAntiPattern,
		Severity:   types.FindingSeverityMedium,
		Confidence: 0.94,
		Title:      "Untracked technology concentration",
		Alternatives: []string{
			"Map the technologies to existing radar entries.",
			"Add a new entry for the missing standard.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectUntrackedEvidence(project, entryByName)
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "A majority of the project's technology surface is still uncharted, so governance decisions are likely missing."
		},
	},
	{
		ID:         "polyglot-spread",
		Kind:       types.FindingKindAntiPattern,
		Severity:   types.FindingSeverityMedium,
		Confidence: 0.90,
		Title:      "Polyglot spread",
		Alternatives: []string{
			"Prefer a single primary runtime per deployable unit.",
			"Document the boundary explicitly if the split is intentional.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectFamilyEvidence(project, 3)
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "This project spans multiple manifest ecosystems, which raises ownership and standardization cost."
		},
	},
	{
		ID:         "ai-mcp-fragmentation",
		Kind:       types.FindingKindAntiPattern,
		Severity:   types.FindingSeverityMedium,
		Confidence: 0.91,
		Title:      "Fragmented AI/MCP tooling",
		Alternatives: []string{
			"Pick one approved SDK or launcher path.",
			"Consolidate config and package entrypoints.",
		},
		Match: func(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
			return collectAIMCPFragmentationEvidence(project)
		},
		Summary: func(project types.Project, evidence []types.FindingEvidence) string {
			return "AI/MCP tooling is spread across multiple signals, which can fragment the approval and rollout path."
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

	// 2. Derive project findings and pattern badges.
	projectSignals := make(map[string]projectSignalResult, len(projects))
	patternProjects := map[string][]string{}
	findings := make([]types.ProjectFinding, 0)
	for _, project := range projects {
		signals := buildProjectSignals(project, entryByName)
		projectSignals[project.ID] = signals
		findings = append(findings, signals.Findings...)
		for _, pattern := range signals.Patterns {
			patternProjects[pattern] = appendUnique(patternProjects[pattern], project.Name)
		}
	}

	// 3. Build edges: technologies that appear together in the same project.
	edges := buildEdges(projects, entryByName)

	// 4. Build per-project stacks.
	stacks := buildStacks(projects, entryByName, projectSignals)

	// 5. Build codebase pattern tags.
	patterns := buildPatternTags(patternProjects)

	// 6. Build file trees.
	fileTrees := buildFileTrees(projects)

	// 7. Summary.
	trackedPercent := 0.0
	if len(techUsage) > 0 {
		trackedPercent = math.Round(float64(trackedCount)/float64(len(techUsage))*1000) / 10
	}

	sortFindings(findings)

	return &types.CodebaseAnalysis{
		Nodes:    nodes,
		Edges:    edges,
		Stacks:   stacks,
		Patterns: patterns,
		Findings: findings,
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

type projectSignalResult struct {
	Patterns []string
	Findings []types.ProjectFinding
}

func buildStacks(projects []types.Project, entryByName map[string]types.Entry, projectSignals map[string]projectSignalResult) []types.ProjectStack {
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
			Patterns:    append([]string(nil), projectSignals[p.ID].Patterns...),
			TechCount:   len(p.Technologies),
		}

		stacks = append(stacks, stack)
	}

	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].TechCount > stacks[j].TechCount
	})

	return stacks
}

func buildProjectSignals(project types.Project, entryByName map[string]types.Entry) projectSignalResult {
	result := projectSignalResult{
		Patterns: make([]string, 0),
		Findings: make([]types.ProjectFinding, 0),
	}

	for _, rule := range projectFindingRules {
		evidence := rule.Match(project, entryByName)
		if len(evidence) == 0 {
			continue
		}

		finding := types.ProjectFinding{
			ID:           findingID(rule.ID, project.ID),
			Kind:         rule.Kind,
			Title:        rule.Title,
			Summary:      rule.Summary(project, evidence),
			Severity:     rule.Severity,
			Confidence:   rule.Confidence,
			Project:      types.ProjectIdentity{ID: project.ID, Name: project.Name},
			Evidence:     evidence,
			Alternatives: append([]string(nil), rule.Alternatives...),
		}
		result.Findings = append(result.Findings, finding)

		if rule.PatternName != "" {
			result.Patterns = appendUnique(result.Patterns, rule.PatternName)
		}
	}

	sort.Strings(result.Patterns)
	return result
}

func buildPatternTags(patternProjects map[string][]string) []types.PatternTag {
	patternByName := make(map[string]projectFindingRule, len(projectFindingRules))
	for _, rule := range projectFindingRules {
		if rule.PatternName == "" {
			continue
		}
		patternByName[rule.PatternName] = rule
	}

	patterns := make([]types.PatternTag, 0, len(patternProjects))
	for name, projects := range patternProjects {
		rule, ok := patternByName[name]
		if !ok {
			continue
		}
		patterns = append(patterns, types.PatternTag{
			Pattern:  name,
			Projects: append([]string(nil), projects...),
			Signal:   rule.Signal,
			Icon:     rule.Icon,
		})
	}

	sort.Slice(patterns, func(i, j int) bool {
		if len(patterns[i].Projects) == len(patterns[j].Projects) {
			return patterns[i].Pattern < patterns[j].Pattern
		}
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

func collectTechEvidence(project types.Project, match func(types.DiscoveredTech) bool) []types.FindingEvidence {
	evidence := make([]types.FindingEvidence, 0)
	seen := map[string]bool{}
	for _, tech := range project.Technologies {
		if !match(tech) {
			continue
		}
		item := findingEvidenceFromTech(tech)
		key := findingEvidenceKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		evidence = append(evidence, item)
	}
	return evidence
}

func techKeywordMatch(tech types.DiscoveredTech, keywords []string) bool {
	fields := []string{
		strings.ToLower(tech.Name),
		strings.ToLower(tech.Source),
		strings.ToLower(tech.Version),
		strings.ToLower(tech.EvidenceTitle),
		strings.ToLower(tech.EvidencePath),
	}
	for _, field := range fields {
		for _, keyword := range keywords {
			if strings.Contains(field, strings.ToLower(keyword)) {
				return true
			}
		}
	}
	return false
}

func sourceExactMatch(source string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.EqualFold(source, candidate) {
			return true
		}
	}
	return false
}

func collectFamilyEvidence(project types.Project, minFamilies int) []types.FindingEvidence {
	familyEvidence := map[string]types.FindingEvidence{}
	for _, tech := range project.Technologies {
		family := sourceFamilyFor(tech)
		if family == "" {
			continue
		}
		if _, ok := familyEvidence[family]; ok {
			continue
		}
		familyEvidence[family] = findingEvidenceFromTech(tech)
	}

	if len(familyEvidence) < minFamilies {
		return nil
	}

	evidence := make([]types.FindingEvidence, 0, len(familyEvidence))
	for _, item := range familyEvidence {
		evidence = append(evidence, item)
	}
	sort.Slice(evidence, func(i, j int) bool {
		return evidence[i].Technology < evidence[j].Technology
	})
	return evidence
}

func collectFullStackEvidence(project types.Project) []types.FindingEvidence {
	frontend := make([]types.FindingEvidence, 0)
	backend := make([]types.FindingEvidence, 0)
	for _, tech := range project.Technologies {
		switch {
		case isFrontendTech(tech):
			frontend = append(frontend, findingEvidenceFromTech(tech))
		case isBackendTech(tech):
			backend = append(backend, findingEvidenceFromTech(tech))
		}
	}
	if len(frontend) == 0 || len(backend) == 0 {
		return nil
	}
	evidence := []types.FindingEvidence{frontend[0], backend[0]}
	return evidence
}

func collectHoldEvidence(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
	evidence := make([]types.FindingEvidence, 0)
	seen := map[string]bool{}
	for _, tech := range project.Technologies {
		entry, ok := entryByName[strings.ToLower(tech.Name)]
		if !ok || entry.Ring != types.RingHold {
			continue
		}
		item := findingEvidenceFromTech(tech)
		key := findingEvidenceKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		evidence = append(evidence, item)
	}
	return evidence
}

func collectUntrackedEvidence(project types.Project, entryByName map[string]types.Entry) []types.FindingEvidence {
	untracked := make([]types.FindingEvidence, 0)
	seen := map[string]bool{}
	for _, tech := range project.Technologies {
		if _, ok := entryByName[strings.ToLower(tech.Name)]; ok {
			continue
		}
		item := findingEvidenceFromTech(tech)
		key := findingEvidenceKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		untracked = append(untracked, item)
	}

	if len(project.Technologies) == 0 || len(untracked) < 3 {
		return nil
	}
	if float64(len(untracked))/float64(len(project.Technologies)) < 0.5 {
		return nil
	}

	sort.Slice(untracked, func(i, j int) bool {
		return untracked[i].Technology < untracked[j].Technology
	})
	if len(untracked) > 4 {
		untracked = untracked[:4]
	}
	return untracked
}

func collectAIMCPFragmentationEvidence(project types.Project) []types.FindingEvidence {
	evidence := make([]types.FindingEvidence, 0)
	seen := map[string]bool{}
	sources := map[string]bool{}
	for _, tech := range project.Technologies {
		if !isAIMCPTech(tech) {
			continue
		}
		item := findingEvidenceFromTech(tech)
		key := findingEvidenceKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		sources[strings.ToLower(tech.Source)] = true
		evidence = append(evidence, item)
	}

	if len(evidence) < 2 {
		return nil
	}
	if len(sources) < 2 && len(evidence) < 3 {
		return nil
	}

	sort.Slice(evidence, func(i, j int) bool {
		return evidence[i].Technology < evidence[j].Technology
	})
	if len(evidence) > 4 {
		evidence = evidence[:4]
	}
	return evidence
}

func findingEvidenceFromTech(tech types.DiscoveredTech) types.FindingEvidence {
	source := tech.Source
	if source == "" {
		source = "unknown"
	}
	title := tech.EvidenceTitle
	if title == "" {
		title = tech.Name
	}
	path := tech.EvidencePath
	if path == "" {
		path = source
	}
	return types.FindingEvidence{
		Technology:    tech.Name,
		Source:        source,
		Version:       tech.Version,
		EvidencePath:  path,
		EvidenceTitle: title,
	}
}

func findingEvidenceKey(evidence types.FindingEvidence) string {
	return strings.ToLower(strings.Join([]string{
		evidence.Technology,
		evidence.Source,
		evidence.Version,
		evidence.EvidencePath,
		evidence.EvidenceTitle,
	}, "|"))
}

func findingID(ruleID, projectID string) string {
	if projectID == "" {
		return ruleID
	}
	return ruleID + ":" + projectID
}

func sortFindings(findings []types.ProjectFinding) {
	severityRank := func(severity types.FindingSeverity) int {
		switch severity {
		case types.FindingSeverityHigh:
			return 0
		case types.FindingSeverityMedium:
			return 1
		default:
			return 2
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Project.Name != findings[j].Project.Name {
			return findings[i].Project.Name < findings[j].Project.Name
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind == types.FindingKindAntiPattern
		}
		if severityRank(findings[i].Severity) != severityRank(findings[j].Severity) {
			return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
		}
		if findings[i].Title != findings[j].Title {
			return findings[i].Title < findings[j].Title
		}
		return findings[i].ID < findings[j].ID
	})
}

func sourceFamilyFor(tech types.DiscoveredTech) string {
	source := strings.ToLower(tech.Source)
	switch source {
	case "go.mod", "go.sum":
		return "go"
	case "package.json", "pnpm-lock.yaml", "yarn.lock", "bun.lockb", "vite.config.ts", "vite.config.js":
		return "javascript"
	case "pyproject.toml", "requirements.txt", "poetry.lock", "setup.py":
		return "python"
	case "cargo.toml":
		return "rust"
	case "pom.xml", "build.gradle", "build.gradle.kts", "gradle.properties":
		return "java"
	case ".csproj", ".sln", "nuget.config":
		return "dotnet"
	case "gemfile", "*.gemspec":
		return "ruby"
	case "composer.json":
		return "php"
	case ".mcp.json", "mcp.json", "claude_desktop_config.json", "cline_mcp_settings.json":
		return "mcp"
	case "dockerfile", "docker-compose.yml", "compose.yaml", "compose.yml":
		return "container"
	case "adr":
		return "techniques"
	}

	switch {
	case strings.HasSuffix(source, ".go"):
		return "go"
	case strings.HasSuffix(source, ".ts"), strings.HasSuffix(source, ".tsx"), strings.HasSuffix(source, ".js"), strings.HasSuffix(source, ".jsx"):
		return "javascript"
	case strings.HasSuffix(source, ".py"):
		return "python"
	case strings.HasSuffix(source, ".rs"):
		return "rust"
	case strings.HasSuffix(source, ".cs"):
		return "dotnet"
	}
	return ""
}

func isFrontendTech(tech types.DiscoveredTech) bool {
	if sourceExactMatch(tech.Source, []string{"package.json", "pnpm-lock.yaml", "yarn.lock", "bun.lockb", "vite.config.ts", "vite.config.js"}) {
		return true
	}
	return techKeywordMatch(tech, []string{
		"react", "react-dom", "next", "nuxt", "vue", "svelte", "angular", "solid", "ember", "astro", "vite", "webpack", "tailwind",
	})
}

func isBackendTech(tech types.DiscoveredTech) bool {
	if sourceExactMatch(tech.Source, []string{"go.mod", "pyproject.toml", "requirements.txt", "poetry.lock", "setup.py", "pom.xml", "build.gradle", "build.gradle.kts", "composer.json"}) {
		return true
	}
	return techKeywordMatch(tech, []string{
		"go", "fastapi", "django", "flask", "express", "nest", "spring", "rails", "laravel", "fiber", "gin", "echo", "actix", "axum", "asp.net", "dotnet",
	})
}

func isAIMCPTech(tech types.DiscoveredTech) bool {
	if sourceExactMatch(tech.Source, []string{".mcp.json", "mcp.json", "claude_desktop_config.json", "cline_mcp_settings.json"}) {
		return true
	}
	return techKeywordMatch(tech, []string{
		"openai", "anthropic", "langchain", "langgraph", "ollama", "vllm", "litellm", "llamaindex", "llama-index", "mcp", "model context protocol", "modelcontextprotocol",
	})
}
