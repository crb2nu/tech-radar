// Package types – Codebase visualizer types.
package types

import "time"

// CodebaseAnalysis is the aggregated response for the codebase visualizer.
type CodebaseAnalysis struct {
	Nodes    []TechNode       `json:"nodes"`
	Edges    []DependencyEdge `json:"edges"`
	Stacks   []ProjectStack   `json:"stacks"`
	Patterns []PatternTag     `json:"patterns"`
	Findings []ProjectFinding `json:"findings,omitempty"`
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
	Ecosystem string `json:"ecosystem"`
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

// FindingKind identifies whether a finding is a positive pattern or an anti-pattern.
type FindingKind string

const (
	FindingKindPattern     FindingKind = "pattern"
	FindingKindAntiPattern FindingKind = "anti_pattern"
)

// FindingSeverity expresses how important a finding is.
type FindingSeverity string

const (
	FindingSeverityLow    FindingSeverity = "low"
	FindingSeverityMedium FindingSeverity = "medium"
	FindingSeverityHigh   FindingSeverity = "high"
)

// ProjectIdentity identifies the project that produced a finding.
type ProjectIdentity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FindingEvidence links a finding back to a concrete scan observation.
type FindingEvidence struct {
	Technology    string `json:"technology"`
	Source        string `json:"source"`
	Version       string `json:"version,omitempty"`
	EvidencePath  string `json:"evidence_path,omitempty"`
	EvidenceTitle string `json:"evidence_title,omitempty"`
}

// ProjectFinding is a per-project architecture signal or warning.
type ProjectFinding struct {
	ID           string            `json:"id"`
	Kind         FindingKind       `json:"kind"`
	Title        string            `json:"title"`
	Summary      string            `json:"summary"`
	Severity     FindingSeverity   `json:"severity"`
	Confidence   float64           `json:"confidence"`
	Project      ProjectIdentity   `json:"project"`
	Evidence     []FindingEvidence `json:"evidence"`
	Alternatives []string          `json:"alternatives,omitempty"`
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
