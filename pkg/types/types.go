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
	Name        string         `yaml:"name" json:"name"`
	Quadrant    Quadrant       `yaml:"quadrant" json:"quadrant"`
	Ring        Ring           `yaml:"ring" json:"ring"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Moved       int            `yaml:"moved" json:"moved"` // -1, 0, 1
	Link        string         `yaml:"link,omitempty" json:"link,omitempty"`
	Tags        []string       `yaml:"tags,omitempty" json:"tags,omitempty"`
	History     []HistoryEntry `yaml:"history,omitempty" json:"history,omitempty"`
}

// HistoryEntry records a change in an entry's position.
type HistoryEntry struct {
	Date   time.Time `yaml:"date" json:"date"`
	Ring   Ring      `yaml:"ring" json:"ring"`
	Reason string    `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// Radar represents the complete tech radar.
type Radar struct {
	Meta    RadarMeta `yaml:"meta" json:"meta"`
	Entries []Entry   `yaml:"entries" json:"entries"`
}

// RadarMeta contains radar metadata.
type RadarMeta struct {
	Name    string    `yaml:"name" json:"name"`
	Updated time.Time `yaml:"updated" json:"updated"`
	Version string    `yaml:"version,omitempty" json:"version,omitempty"`
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
