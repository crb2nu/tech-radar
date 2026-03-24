package codebase

import (
	"testing"
	"time"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestAnalyze_Empty(t *testing.T) {
	result := Analyze(nil, nil)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(result.Edges))
	}
	if result.Summary.TotalProjects != 0 {
		t.Errorf("expected 0 projects, got %d", result.Summary.TotalProjects)
	}
}

func TestAnalyze_NodesAndEdges(t *testing.T) {
	radar := &types.Radar{
		Entries: []types.Entry{
			{Name: "Go", Quadrant: types.QuadrantLanguages, Ring: types.RingAdopt},
			{Name: "React", Quadrant: types.QuadrantFrameworks, Ring: types.RingAdopt},
		},
	}

	projects := []types.Project{
		{
			ID:   "1",
			Name: "project-alpha",
			Technologies: []types.DiscoveredTech{
				{Name: "Go", Source: "go.mod"},
				{Name: "React", Source: "package.json"},
			},
		},
		{
			ID:   "2",
			Name: "project-beta",
			Technologies: []types.DiscoveredTech{
				{Name: "Go", Source: "go.mod"},
				{Name: "cobra", Source: "go.mod"},
			},
		},
	}

	result := Analyze(radar, projects)

	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}

	// Go should be used by 2 projects
	var goNode *types.TechNode
	for i, n := range result.Nodes {
		if n.Name == "Go" {
			goNode = &result.Nodes[i]
			break
		}
	}
	if goNode == nil {
		t.Fatal("expected Go node")
	}
	if goNode.Usage != 2 {
		t.Errorf("Go usage: expected 2, got %d", goNode.Usage)
	}
	if !goNode.IsTracked {
		t.Error("Go should be tracked")
	}

	// There should be an edge between Go and React (from alpha) and Go and cobra (from beta).
	if len(result.Edges) == 0 {
		t.Fatal("expected at least one edge")
	}

	if result.Summary.TotalProjects != 2 {
		t.Errorf("expected 2 projects, got %d", result.Summary.TotalProjects)
	}
	if result.Summary.TotalTechnologies != 3 {
		t.Errorf("expected 3 technologies, got %d", result.Summary.TotalTechnologies)
	}
}

func TestAnalyze_PatternDetection(t *testing.T) {
	radar := &types.Radar{}
	projects := []types.Project{
		{
			ID:   "1",
			Name: "gitops-project",
			Technologies: []types.DiscoveredTech{
				{Name: "Flux", Source: "go.mod"},
				{Name: "kustomize", Source: "go.mod"},
			},
		},
		{
			ID:   "2",
			Name: "mcp-project",
			Technologies: []types.DiscoveredTech{
				{Name: "Model Context Protocol", Source: ".mcp.json"},
				{Name: "mcp-go", Source: "go.mod"},
			},
		},
		{
			ID:   "3",
			Name: "ai-project",
			Technologies: []types.DiscoveredTech{
				{Name: "langchain", Source: "pyproject.toml"},
				{Name: "openai", Source: "pyproject.toml"},
			},
		},
	}

	result := Analyze(radar, projects)

	patternNames := map[string]bool{}
	for _, p := range result.Patterns {
		patternNames[p.Pattern] = true
	}

	if !patternNames["GitOps"] {
		t.Error("expected GitOps pattern to be detected")
	}
	if !patternNames["MCP-enabled"] {
		t.Error("expected MCP-enabled pattern to be detected")
	}
	if !patternNames["AI/ML pipeline"] {
		t.Error("expected AI/ML pipeline pattern to be detected")
	}
}

func TestAnalyze_FileTree(t *testing.T) {
	radar := &types.Radar{}
	projects := []types.Project{
		{
			ID:   "1",
			Name: "test-project",
			Technologies: []types.DiscoveredTech{
				{Name: "Go", Source: "go.mod"},
				{Name: "cobra", Source: "go.mod"},
				{Name: "React", Source: "package.json"},
			},
		},
	}

	result := Analyze(radar, projects)

	if len(result.FileTree) != 1 {
		t.Fatalf("expected 1 project tree, got %d", len(result.FileTree))
	}

	tree := result.FileTree[0]
	if tree.ProjectName != "test-project" {
		t.Errorf("expected project name 'test-project', got %q", tree.ProjectName)
	}
	if len(tree.Files) != 2 {
		t.Errorf("expected 2 file nodes, got %d", len(tree.Files))
	}
}

func TestAnalyze_Stacks(t *testing.T) {
	radar := &types.Radar{
		Entries: []types.Entry{
			{Name: "Go", Quadrant: types.QuadrantLanguages, Ring: types.RingAdopt},
			{Name: "React", Quadrant: types.QuadrantFrameworks, Ring: types.RingAdopt},
			{Name: "Docker", Quadrant: types.QuadrantTools, Ring: types.RingAdopt},
		},
	}
	projects := []types.Project{
		{
			ID:       "1",
			Name:     "fullstack-app",
			LastScan: time.Now(),
			Technologies: []types.DiscoveredTech{
				{Name: "Go", Source: "go.mod"},
				{Name: "React", Source: "package.json"},
				{Name: "Docker", Source: "go.mod"},
			},
		},
	}

	result := Analyze(radar, projects)

	if len(result.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(result.Stacks))
	}

	stack := result.Stacks[0]
	if stack.ProjectName != "fullstack-app" {
		t.Errorf("expected project name 'fullstack-app', got %q", stack.ProjectName)
	}

	langs := stack.Categories["languages"]
	if len(langs) == 0 {
		t.Error("expected languages category to contain Go")
	}

	frameworks := stack.Categories["frameworks"]
	if len(frameworks) == 0 {
		t.Error("expected frameworks category to contain React")
	}
}

func TestAnalyze_TrackedPercent(t *testing.T) {
	radar := &types.Radar{
		Entries: []types.Entry{
			{Name: "Go", Quadrant: types.QuadrantLanguages, Ring: types.RingAdopt},
		},
	}
	projects := []types.Project{
		{
			ID:   "1",
			Name: "p1",
			Technologies: []types.DiscoveredTech{
				{Name: "Go", Source: "go.mod"},
				{Name: "untracked-lib", Source: "go.mod"},
			},
		},
	}

	result := Analyze(radar, projects)
	if result.Summary.TrackedPercent != 50.0 {
		t.Errorf("expected 50.0%% tracked, got %.1f%%", result.Summary.TrackedPercent)
	}
}

func TestAnalyze_MonorepoPattern(t *testing.T) {
	radar := &types.Radar{}
	projects := []types.Project{
		{
			ID:   "1",
			Name: "mono-project",
			Technologies: []types.DiscoveredTech{
				{Name: "Go", Source: "go.mod"},
				{Name: "React", Source: "package.json"},
				{Name: "FastAPI", Source: "pyproject.toml"},
			},
		},
	}

	result := Analyze(radar, projects)
	patternNames := map[string]bool{}
	for _, p := range result.Patterns {
		patternNames[p.Pattern] = true
	}
	if !patternNames["Monorepo"] {
		t.Error("expected Monorepo pattern for multi-language project")
	}
}
