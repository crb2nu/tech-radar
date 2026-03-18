// Package types defines data structures for Tech Radar.
package types

import "time"

// Quadrant represents a section of the radar.
type Quadrant string

const (
	QuadrantLanguages  Quadrant = "languages"
	QuadrantFrameworks Quadrant = "frameworks"
	QuadrantTools      Quadrant = "tools"
	QuadrantPlatforms  Quadrant = "platforms"
	QuadrantTechniques Quadrant = "techniques"
)

// Ring represents the adoption level.
type Ring string

const (
	RingAdopt  Ring = "adopt"
	RingTrial  Ring = "trial"
	RingAssess Ring = "assess"
	RingHold   Ring = "hold"
)

// Entry represents a single technology on the radar.
type Entry struct {
	ID          string         `yaml:"id" json:"id"`
	Name        string         `yaml:"name" json:"name"`
	Quadrant    Quadrant       `yaml:"quadrant" json:"quadrant"`
	Ring        Ring           `yaml:"ring" json:"ring"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Moved       int            `yaml:"moved" json:"moved"` // -1, 0, 1
	Link        string         `yaml:"link,omitempty" json:"link,omitempty"`
	Tags        []string       `yaml:"tags,omitempty" json:"tags,omitempty"`
	History     []HistoryEntry `yaml:"history,omitempty" json:"history,omitempty"`
	ProjectIDs  []string       `yaml:"project_ids,omitempty" json:"project_ids,omitempty"`
}

// HistoryEntry records a change in an entry's position.
type HistoryEntry struct {
	Date   time.Time `yaml:"date" json:"date"`
	Ring   Ring      `yaml:"ring" json:"ring"`
	Reason string    `yaml:"reason,omitempty" json:"reason,omitempty"`
	Author string    `yaml:"author,omitempty" json:"author,omitempty"`
}

// Radar represents the complete tech radar.
type Radar struct {
	ID      string    `yaml:"id" json:"id"`
	Meta    RadarMeta `yaml:"meta" json:"meta"`
	Entries []Entry   `yaml:"entries" json:"entries"`
}

// RadarMeta contains radar metadata.
type RadarMeta struct {
	Name    string    `yaml:"name" json:"name"`
	Updated time.Time `yaml:"updated" json:"updated"`
	Version string    `yaml:"version,omitempty" json:"version,omitempty"`
}

// Project represents a scanned repository or application.
type Project struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	URL          string           `json:"url,omitempty"`
	LastScan     time.Time        `json:"last_scan"`
	Technologies []DiscoveredTech `json:"technologies"`
}

// APIKey represents a machine token for ingestion.
type APIKey struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"` // Hashed in DB
	Name      string    `json:"name"`
	ProjectID string    `json:"project_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Config holds scanner and visualization configuration.
type Config struct {
	Scan          ScanConfig          `yaml:"scan" json:"scan"`
	Visualization VisualizationConfig `yaml:"visualization" json:"visualization"`
}

// ScanConfig configures the project scanner.
type ScanConfig struct {
	Files    []string             `yaml:"files" json:"files"`
	Mappings map[string]EntryHint `yaml:"mappings" json:"mappings"`
	Exclude  []string             `yaml:"exclude" json:"exclude"`
}

// EntryHint provides hints for auto-discovered entries.
type EntryHint struct {
	Name     string   `yaml:"name" json:"name"`
	Quadrant Quadrant `yaml:"quadrant" json:"quadrant"`
}

// VisualizationConfig configures the radar visualization.
type VisualizationConfig struct {
	Colors   map[string]string `yaml:"colors" json:"colors"`
	Title    string            `yaml:"title" json:"title"`
	ShowDots bool              `yaml:"show_dots" json:"show_dots"`
}

// ScanResult represents technologies found in a project.
type ScanResult struct {
	ProjectID    string              `json:"project_id"`
	Path         string              `json:"path"`
	Technologies []DiscoveredTech    `json:"technologies"`
	Files        map[string][]string `json:"files"` // File -> deps
}

// DiscoveredTech represents a technology found during scanning.
type DiscoveredTech struct {
	Name     string   `json:"name"`
	Source   string   `json:"source"` // package.json, go.mod, etc.
	Version  string   `json:"version,omitempty"`
	Quadrant Quadrant `json:"quadrant,omitempty"` // If known from mappings
}

// Insights represents aggregated analytics for the ecosystem.
type Insights struct {
	TopUncharted      []UsageStat     `json:"top_uncharted"`
	TopAdopted        []UsageStat     `json:"top_adopted"`
	TopDrift          []UsageStat     `json:"top_drift"`
	TopAIMCPUncharted []UsageStat     `json:"top_ai_mcp_uncharted"`
	TopAIMCPTracked   []UsageStat     `json:"top_ai_mcp_tracked"`
	TopTags           []UsageStat     `json:"top_tags"`
	QuadrantBreakdown []LabelStat     `json:"quadrant_breakdown"`
	RingBreakdown     []LabelStat     `json:"ring_breakdown"`
	SourceBreakdown   []LabelStat     `json:"source_breakdown"`
	RecentMoves       []RecentMove    `json:"recent_moves"`
	ProjectHealth     []ProjectHealth `json:"project_health"`
	Summary           InsightSummary  `json:"summary"`
	ProjectsCount     int             `json:"projects_count"`
	TechCount         int             `json:"tech_count"`
}

// UsageStat represents the usage count of a technology.
type UsageStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// LabelStat represents a categorical count.
type LabelStat struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// RecentMove captures the latest movement for a tracked technology.
type RecentMove struct {
	Name         string    `json:"name"`
	FromRing     string    `json:"from_ring,omitempty"`
	ToRing       string    `json:"to_ring"`
	Date         time.Time `json:"date"`
	Reason       string    `json:"reason,omitempty"`
	Author       string    `json:"author,omitempty"`
	ProjectCount int       `json:"project_count"`
}

// ProjectHealth summarizes alignment between a project and the radar.
type ProjectHealth struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	LastScan            time.Time `json:"last_scan"`
	TotalTechs          int       `json:"total_techs"`
	TrackedTechs        int       `json:"tracked_techs"`
	UntrackedTechs      int       `json:"untracked_techs"`
	AdoptTechs          int       `json:"adopt_techs"`
	TrialTechs          int       `json:"trial_techs"`
	AssessTechs         int       `json:"assess_techs"`
	HoldTechs           int       `json:"hold_techs"`
	CoveragePercent     float64   `json:"coverage_percent"`
	RadarAlignmentScore int       `json:"radar_alignment_score"`
	RiskLevel           string    `json:"risk_level"`
	TopTechs            []string  `json:"top_techs"`
}

// InsightSummary provides headline metrics for the dashboard.
type InsightSummary struct {
	RadarCoveragePercent float64 `json:"radar_coverage_percent"`
	TrackedUsageCount    int     `json:"tracked_usage_count"`
	UnchartedUsageCount  int     `json:"uncharted_usage_count"`
	ActiveDriftCount     int     `json:"active_drift_count"`
	RecentDecisionCount  int     `json:"recent_decision_count"`
	AIMCPProjectCount    int     `json:"ai_mcp_project_count"`
	AIMCPUnchartedCount  int     `json:"ai_mcp_uncharted_count"`
}
