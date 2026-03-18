package db

import (
	"database/sql"
	"fmt"

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

	if err == sql.ErrNoRows {
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
	defer rows.Close()

	for rows.Next() {
		var e types.Entry
		err := rows.Scan(&e.ID, &e.Name, &e.Quadrant, &e.Ring, &e.Description, &e.Moved, &e.Link, (*pq.StringArray)(&e.Tags))
		if err != nil {
			return nil, err
		}

		// Fetch history for each entry
		hRows, err := d.pool.Query(`
			SELECT date, ring, reason, author FROM entry_history 
			WHERE entry_id = $1 ORDER BY date DESC
		`, e.ID)
		if err == nil {
			for hRows.Next() {
				var h types.HistoryEntry
				if err := hRows.Scan(&h.Date, &h.Ring, &h.Reason, &h.Author); err == nil {
					e.History = append(e.History, h)
				}
			}
			hRows.Close()
		}

		// Fetch projects using this tech
		pRows, err := d.pool.Query(`
			SELECT p.name FROM projects p
			JOIN project_technologies pt ON p.id = pt.project_id
			WHERE pt.tech_name = $1
		`, e.Name)
		if err == nil {
			for pRows.Next() {
				var pName string
				if err := pRows.Scan(&pName); err == nil {
					e.ProjectIDs = append(e.ProjectIDs, pName)
				}
			}
			pRows.Close()
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
	defer rows.Close()

	var projects []types.Project
	for rows.Next() {
		var p types.Project
		var url sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &url, &p.LastScan); err != nil {
			return nil, err
		}
		p.URL = url.String

		// Fetch technologies for this project
		tRows, err := d.pool.Query("SELECT tech_name, version, source FROM project_technologies WHERE project_id = $1", p.ID)
		if err == nil {
			for tRows.Next() {
				var t types.DiscoveredTech
				if err := tRows.Scan(&t.Name, &t.Version, &t.Source); err == nil {
					p.Technologies = append(p.Technologies, t)
				}
			}
			tRows.Close()
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
	defer tx.Rollback()

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

	// 3. Insert new project_technologies
	for _, tech := range result.Technologies {
		_, err = tx.Exec(`
			INSERT INTO project_technologies (project_id, tech_name, version, source)
			VALUES ($1, $2, $3, $4)
		`, projectID, tech.Name, tech.Version, tech.Source)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
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

	if err == sql.ErrNoRows {
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
	defer tx.Rollback()

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

	if err == sql.ErrNoRows {
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
