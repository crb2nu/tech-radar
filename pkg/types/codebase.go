// Package types – Codebase visualizer types.
package types

import "time"

// CodebaseAnalysis is the aggregated response for the codebase visualizer.
type CodebaseAnalysis struct {
	Nodes    []TechNode       `json:"nodes"`
	Edges    []DependencyEdge `json:"edges"`
	Stacks   []ProjectStack   `json:"stacks"`
	Patterns []PatternTag     `json:"patterns"`
	FileTree []ProjectTree    `json:"file_tree"`
	Summary  CodebaseSummary  `json:"summary"`
}

// TechNode represents a technology as a graph node.
type TechNode struct {
	Name      string `json:"name"`
	Quadrant  string `json:"quadrant"`
	Ring      string `json:"ring"`
	Usage     int    `json:"usage"`
	IsTracked bool   `json:"is_tracked"`
}

// DependencyEdge represents a co-usage relationship between two technologies.
type DependencyEdge struct {
	Source   string   `json:"source"`
	Target   string   `json:"target"`
	Weight   int      `json:"weight"`
	Projects []string `json:"projects"`
}

// ProjectStack represents one project's technology breakdown by category.
type ProjectStack struct {
	ProjectID   string              `json:"project_id"`
	ProjectName string              `json:"project_name"`
	LastScan    time.Time           `json:"last_scan"`
	Categories  map[string][]string `json:"categories"`
	Patterns    []string            `json:"patterns"`
	TechCount   int                 `json:"tech_count"`
}

// PatternTag is a heuristic-detected architectural pattern.
type PatternTag struct {
	Pattern  string   `json:"pattern"`
	Projects []string `json:"projects"`
	Signal   string   `json:"signal"`
	Icon     string   `json:"icon"`
}

// ProjectTree is a per-project file→technologies mapping.
type ProjectTree struct {
	ProjectID   string     `json:"project_id"`
	ProjectName string     `json:"project_name"`
	Files       []FileNode `json:"files"`
}

// FileNode is one scanned manifest file and its discovered technologies.
type FileNode struct {
	Path         string   `json:"path"`
	Technologies []string `json:"technologies"`
}

// CodebaseSummary provides headline numbers for the visualizer.
type CodebaseSummary struct {
	TotalProjects     int     `json:"total_projects"`
	TotalTechnologies int     `json:"total_technologies"`
	TotalEdges        int     `json:"total_edges"`
	TotalPatterns     int     `json:"total_patterns"`
	TrackedPercent    float64 `json:"tracked_percent"`
}
