package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

// DB handles database operations.
type DB struct {
	pool *sql.DB
}

// New creates a new DB connection.
func New(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{pool: db}, nil
}

// GetRadar fetches a radar by ID.
func (d *DB) GetRadar(id string) (*types.Radar, error) {
	var radar types.Radar
	var err error
	var version sql.NullString

	if id == "default" {
		err = d.pool.QueryRow(`
			SELECT id, name, updated_at, version 
			FROM radars WHERE name = 'default' OR name = 'Enterprise Technology Radar'
			ORDER BY name DESC
			LIMIT 1
		`).Scan(&radar.ID, &radar.Meta.Name, &radar.Meta.Updated, &version)
	} else {
		err = d.pool.QueryRow(`
			SELECT id, name, updated_at, version 
			FROM radars WHERE id = $1
		`, id).Scan(&radar.ID, &radar.Meta.Name, &radar.Meta.Updated, &version)
	}
	radar.Meta.Version = version.String

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("radar not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	// Fetch entries
	rows, err := d.pool.Query(`
		SELECT id, name, quadrant, ring, description, moved, link, tags
		FROM entries WHERE radar_id = $1
	`, radar.ID)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var e types.Entry
		var desc, link sql.NullString
		err := rows.Scan(&e.ID, &e.Name, &e.Quadrant, &e.Ring, &desc, &e.Moved, &link, (*pq.StringArray)(&e.Tags))
		if err != nil {
			return nil, err
		}
		e.Description = desc.String
		e.Link = link.String

		// Fetch history for each entry
		hRows, err := d.pool.Query(`
			SELECT date, ring, reason, author FROM entry_history 
			WHERE entry_id = $1 ORDER BY date DESC
		`, e.ID)
		if err == nil {
			for hRows.Next() {
				var h types.HistoryEntry
				var reason, author sql.NullString
				if err := hRows.Scan(&h.Date, &h.Ring, &reason, &author); err == nil {
					h.Reason = reason.String
					h.Author = author.String
					e.History = append(e.History, h)
				}
			}
			closeRows(hRows)
		}

		// Fetch projects using this tech
		pRows, err := d.pool.Query(`
			SELECT DISTINCT p.name FROM projects p
			JOIN project_technologies pt ON p.id = pt.project_id
			WHERE pt.tech_name = $1
			ORDER BY p.name ASC
		`, e.Name)
		if err == nil {
			for pRows.Next() {
				var pName string
				if err := pRows.Scan(&pName); err == nil {
					e.ProjectIDs = append(e.ProjectIDs, pName)
				}
			}
			closeRows(pRows)
		}

		radar.Entries = append(radar.Entries, e)
	}

	return &radar, nil
}

// GetProjects fetches all projects and their technologies.
func (d *DB) GetProjects() ([]types.Project, error) {
	rows, err := d.pool.Query("SELECT id, name, url, last_scan FROM projects ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer closeRows(rows)

	var projects []types.Project
	for rows.Next() {
		var p types.Project
		var url sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &url, &p.LastScan); err != nil {
			return nil, err
		}
		p.URL = url.String

		// Fetch summarized technologies for this project, including persisted provenance.
		tRows, err := d.pool.Query(`
			SELECT tech_name, version, source, evidence_path, evidence_title
			FROM project_technologies
			WHERE project_id = $1
			ORDER BY tech_name ASC, COALESCE(source, '') ASC, COALESCE(evidence_path, '') ASC
		`, p.ID)
		if err == nil {
			for tRows.Next() {
				var t types.DiscoveredTech
				var version sql.NullString
				var source sql.NullString
				var evidencePath sql.NullString
				var evidenceTitle sql.NullString
				if err := tRows.Scan(&t.Name, &version, &source, &evidencePath, &evidenceTitle); err == nil {
					t.Version = version.String
					t.Source = source.String
					t.EvidencePath = evidencePath.String
					t.EvidenceTitle = evidenceTitle.String
					p.Technologies = append(p.Technologies, t)
				}
			}
			closeRows(tRows)
		}

		// Fetch raw observations so the API can preserve multiple evidence rows per technology.
		oRows, err := d.pool.Query(`
			SELECT tech_name, version, source, evidence_path, evidence_title, observation_kind, observed_at
			FROM project_technology_observations
			WHERE project_id = $1
			ORDER BY observed_at DESC, tech_name ASC, COALESCE(source, '') ASC
		`, p.ID)
		if err == nil {
			for oRows.Next() {
				var obs types.DiscoveredTech
				var version sql.NullString
				var source sql.NullString
				var evidencePath sql.NullString
				var evidenceTitle sql.NullString
				var observationKind sql.NullString
				var observedAt time.Time
				if err := oRows.Scan(&obs.Name, &version, &source, &evidencePath, &evidenceTitle, &observationKind, &observedAt); err == nil {
					obs.Version = version.String
					obs.Source = source.String
					obs.EvidencePath = evidencePath.String
					obs.EvidenceTitle = evidenceTitle.String
					obs.ObservationKind = observationKind.String
					observedAtCopy := observedAt
					obs.ObservedAt = &observedAtCopy
					p.Observations = append(p.Observations, obs)
				}
			}
			closeRows(oRows)
		}

		projects = append(projects, p)
	}

	return projects, nil
}

// GetInsights calculates ecosystem-wide technology adoption analytics.
func (d *DB) GetInsights() (*types.Insights, error) {
	radar, err := d.GetRadar("default")
	if err != nil {
		radar = &types.Radar{}
	}

	projects, err := d.GetProjects()
	if err != nil {
		return nil, err
	}

	return buildInsights(radar, projects), nil
}

// SaveScan records a project scan result.
func (d *DB) SaveScan(result types.ScanResult) error {
	tx, err := d.pool.Begin()
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	// 1. Ensure project exists or update last_scan
	var projectID string
	err = tx.QueryRow(`
		INSERT INTO projects (name, last_scan) 
		VALUES ($1, CURRENT_TIMESTAMP)
		ON CONFLICT (name) DO UPDATE SET last_scan = CURRENT_TIMESTAMP
		RETURNING id
	`, result.ProjectID).Scan(&projectID)
	if err != nil {
		// Try to get existing ID if conflict didn't return it (depends on PG version/config)
		err = tx.QueryRow("SELECT id FROM projects WHERE name = $1", result.ProjectID).Scan(&projectID)
		if err != nil {
			return err
		}
	}

	// 2. Clear old project_technologies
	_, err = tx.Exec("DELETE FROM project_technologies WHERE project_id = $1", projectID)
	if err != nil {
		return err
	}

	// 3. Clear raw observations.
	_, err = tx.Exec("DELETE FROM project_technology_observations WHERE project_id = $1", projectID)
	if err != nil {
		return err
	}

	observations := projectObservations(result)
	summaryTechnologies := summarizeTechnologies(result, observations)

	// 4. Insert summarized technologies for compatibility with existing consumers.
	for _, tech := range summaryTechnologies {
		_, err = tx.Exec(`
			INSERT INTO project_technologies (project_id, tech_name, version, source, evidence_path, evidence_title)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, projectID, tech.Name, tech.Version, tech.Source, tech.EvidencePath, tech.EvidenceTitle)
		if err != nil {
			return err
		}
	}

	// 5. Insert the full observation stream so duplicate tech sightings are retained.
	for _, obs := range observations {
		var observedAt any
		if obs.ObservedAt != nil {
			observedAt = *obs.ObservedAt
		}

		_, err = tx.Exec(`
			INSERT INTO project_technology_observations (
				project_id, tech_name, version, source, evidence_path, evidence_title, observation_kind, observed_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, CURRENT_TIMESTAMP))
		`, projectID, obs.Name, obs.Version, obs.Source, obs.EvidencePath, obs.EvidenceTitle, obs.ObservationKind, observedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func projectObservations(result types.ScanResult) []types.DiscoveredTech {
	if len(result.Observations) > 0 {
		return append([]types.DiscoveredTech(nil), result.Observations...)
	}

	return append([]types.DiscoveredTech(nil), result.Technologies...)
}

func summarizeTechnologies(result types.ScanResult, observations []types.DiscoveredTech) []types.DiscoveredTech {
	if len(result.Technologies) > 0 {
		return append([]types.DiscoveredTech(nil), result.Technologies...)
	}

	return summarizeObservations(observations)
}

func summarizeObservations(observations []types.DiscoveredTech) []types.DiscoveredTech {
	summaries := make([]types.DiscoveredTech, 0, len(observations))
	seen := make(map[string]int, len(observations))

	for _, obs := range observations {
		key := strings.ToLower(strings.TrimSpace(obs.Name))
		if key == "" {
			continue
		}

		if idx, ok := seen[key]; ok {
			summaries[idx] = mergeObservationSummary(summaries[idx], obs)
			continue
		}

		seen[key] = len(summaries)
		summaries = append(summaries, obs)
	}

	return summaries
}

func mergeObservationSummary(existing, next types.DiscoveredTech) types.DiscoveredTech {
	if existing.Name == "" {
		existing.Name = next.Name
	}
	if existing.Source == "" {
		existing.Source = next.Source
	}
	if existing.Version == "" {
		existing.Version = next.Version
	}
	if existing.Quadrant == "" {
		existing.Quadrant = next.Quadrant
	}
	if existing.EvidencePath == "" {
		existing.EvidencePath = next.EvidencePath
	}
	if existing.EvidenceTitle == "" {
		existing.EvidenceTitle = next.EvidenceTitle
	}
	if existing.ObservationKind == "" {
		existing.ObservationKind = next.ObservationKind
	}
	if existing.ObservedAt == nil {
		existing.ObservedAt = next.ObservedAt
	}
	return existing
}

// HasAPIKeys reports whether ingestion auth should be enforced.
func (d *DB) HasAPIKeys() (bool, error) {
	var exists bool
	err := d.pool.QueryRow(`SELECT EXISTS (SELECT 1 FROM api_keys LIMIT 1)`).Scan(&exists)
	return exists, err
}

// ValidateAPIKey checks whether the provided key matches a stored API key hash.
func (d *DB) ValidateAPIKey(apiKey string) (bool, error) {
	var exists bool
	err := d.pool.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM api_keys WHERE key_hash = $1 LIMIT 1
		)
	`, hashAPIKey(apiKey)).Scan(&exists)
	return exists, err
}

func hashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

// UpsertEntry adds or updates a radar entry and records history.
func (d *DB) UpsertEntry(entry types.Entry) error {
	// 1. Ensure radar exists (assume default)
	var radarID string
	err := d.pool.QueryRow(`
		INSERT INTO radars (name) VALUES ('default')
		ON CONFLICT DO NOTHING
		RETURNING id
	`).Scan(&radarID)

	if errors.Is(err, sql.ErrNoRows) {
		// Already exists, fetch it
		err = d.pool.QueryRow("SELECT id FROM radars WHERE name = 'default'").Scan(&radarID)
	}
	if err != nil {
		return fmt.Errorf("failed to ensure radar: %w", err)
	}

	tx, err := d.pool.Begin()
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	// 2. Check if entry exists
	var existingID string
	var existingRing string
	err = tx.QueryRow("SELECT id, ring FROM entries WHERE name = $1 AND radar_id = $2", entry.Name, radarID).Scan(&existingID, &existingRing)

	// Calculate movement vector
	movement := 0
	if existingRing != "" && existingRing != string(entry.Ring) {
		rings := map[string]int{"adopt": 4, "trial": 3, "assess": 2, "hold": 1}
		if rings[string(entry.Ring)] > rings[existingRing] {
			movement = 1 // Promotion
		} else {
			movement = -1 // Demotion
		}
	}

	if errors.Is(err, sql.ErrNoRows) {
		// Insert new entry
		err = tx.QueryRow(`
			INSERT INTO entries (radar_id, name, quadrant, ring, description, moved, link, tags)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`, radarID, entry.Name, entry.Quadrant, entry.Ring, entry.Description, movement, entry.Link, pq.StringArray(entry.Tags)).Scan(&existingID)
	} else if err == nil {
		// Update existing entry
		_, err = tx.Exec(`
			UPDATE entries SET 
				quadrant = $1, ring = $2, description = $3, moved = $4, link = $5, tags = $6, updated_at = CURRENT_TIMESTAMP
			WHERE id = $7
		`, entry.Quadrant, entry.Ring, entry.Description, movement, entry.Link, pq.StringArray(entry.Tags), existingID)
	}

	if err != nil {
		return err
	}

	// 3. Add to history if ring changed or new entry
	if existingRing != string(entry.Ring) {
		_, err = tx.Exec(`
			INSERT INTO entry_history (entry_id, ring, reason, author)
			VALUES ($1, $2, $3, $4)
		`, existingID, entry.Ring, entry.Description, "admin")
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.pool.Close()
}

func closeRows(rows *sql.Rows) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		log.Printf("Error closing rows: %v", err)
	}
}

func rollbackTx(tx *sql.Tx) {
	if tx == nil {
		return
	}
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		log.Printf("Error rolling back transaction: %v", err)
	}
}
