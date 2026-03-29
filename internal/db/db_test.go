package db

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestSaveScanPersistsObservationStream(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	database := &DB{pool: mockDB}
	observedAtManifest := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	observedAtADR := time.Date(2026, 3, 19, 12, 5, 0, 0, time.UTC)
	observedAtGo := time.Date(2026, 3, 19, 12, 10, 0, 0, time.UTC)

	result := types.ScanResult{
		ProjectID: "alpha",
		Observations: []types.DiscoveredTech{
			{
				Name:            "React",
				Version:         "^19.0.0",
				Source:          "package.json",
				EvidencePath:    "web/package.json",
				ObservationKind: "manifest",
				ObservedAt:      &observedAtManifest,
			},
			{
				Name:            "React",
				Version:         "^19.0.0",
				Source:          "adr",
				EvidencePath:    "docs/adrs/0007-adopt-react.md",
				EvidenceTitle:   "ADR-0007 Adopt React",
				ObservationKind: "adr",
				ObservedAt:      &observedAtADR,
			},
			{
				Name:            "Go",
				Version:         "1.24.0",
				Source:          "go.mod",
				EvidencePath:    "go.mod",
				ObservationKind: "manifest",
				ObservedAt:      &observedAtGo,
			},
		},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO projects (name, last_scan) 
		VALUES ($1, CURRENT_TIMESTAMP)
		ON CONFLICT (name) DO UPDATE SET last_scan = CURRENT_TIMESTAMP
		RETURNING id
	`)).
		WithArgs("alpha").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("project-1"))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM project_technologies WHERE project_id = $1")).
		WithArgs("project-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM project_technology_observations WHERE project_id = $1")).
		WithArgs("project-1").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO project_technologies (project_id, tech_name, version, source, evidence_path, evidence_title)
			VALUES ($1, $2, $3, $4, $5, $6)
		`)).
		WithArgs("project-1", "React", "^19.0.0", "package.json", "web/package.json", "ADR-0007 Adopt React").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO project_technologies (project_id, tech_name, version, source, evidence_path, evidence_title)
			VALUES ($1, $2, $3, $4, $5, $6)
		`)).
		WithArgs("project-1", "Go", "1.24.0", "go.mod", "go.mod", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO project_technology_observations (
				project_id, tech_name, version, source, evidence_path, evidence_title, observation_kind, observed_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, CURRENT_TIMESTAMP))
		`)).
		WithArgs("project-1", "React", "^19.0.0", "package.json", "web/package.json", "", "manifest", observedAtManifest).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO project_technology_observations (
				project_id, tech_name, version, source, evidence_path, evidence_title, observation_kind, observed_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, CURRENT_TIMESTAMP))
		`)).
		WithArgs("project-1", "React", "^19.0.0", "adr", "docs/adrs/0007-adopt-react.md", "ADR-0007 Adopt React", "adr", observedAtADR).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO project_technology_observations (
				project_id, tech_name, version, source, evidence_path, evidence_title, observation_kind, observed_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, CURRENT_TIMESTAMP))
		`)).
		WithArgs("project-1", "Go", "1.24.0", "go.mod", "go.mod", "", "manifest", observedAtGo).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := database.SaveScan(result); err != nil {
		t.Fatalf("SaveScan returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestGetProjectsReturnsEvidenceFieldsAndObservations(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	database := &DB{pool: mockDB}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	older := now.Add(-5 * time.Minute)

	projectRows := sqlmock.NewRows([]string{"id", "name", "url", "last_scan"}).
		AddRow("project-1", "alpha", "https://example.com/alpha", now)
	techRows := sqlmock.NewRows([]string{"tech_name", "version", "source", "evidence_path", "evidence_title"}).
		AddRow("React", "^19.0.0", "package.json", "web/package.json", "Package manifest")
	observationRows := sqlmock.NewRows([]string{"tech_name", "version", "source", "evidence_path", "evidence_title", "observation_kind", "observed_at"}).
		AddRow("React", "^19.0.0", "adr", "docs/adrs/0007-adopt-react.md", "ADR-0007 Adopt React", "adr", now).
		AddRow("React", "^19.0.0", "package.json", "web/package.json", "Package manifest", "manifest", older)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, url, last_scan FROM projects ORDER BY name ASC")).
		WillReturnRows(projectRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT tech_name, version, source, evidence_path, evidence_title
		FROM project_technologies
		WHERE project_id = $1
		ORDER BY tech_name ASC, COALESCE(source, '') ASC, COALESCE(evidence_path, '') ASC
	`)).
		WithArgs("project-1").
		WillReturnRows(techRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT tech_name, version, source, evidence_path, evidence_title, observation_kind, observed_at
		FROM project_technology_observations
		WHERE project_id = $1
		ORDER BY observed_at DESC, tech_name ASC, COALESCE(source, '') ASC
	`)).
		WithArgs("project-1").
		WillReturnRows(observationRows)

	projects, err := database.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects returned error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if len(projects[0].Technologies) != 1 {
		t.Fatalf("expected 1 summarized technology, got %d", len(projects[0].Technologies))
	}
	if len(projects[0].Observations) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(projects[0].Observations))
	}

	tech := projects[0].Technologies[0]
	assertProjectTechField(t, "name", tech.Name, "React")
	assertProjectTechField(t, "source", tech.Source, "package.json")
	assertProjectTechField(t, "version", tech.Version, "^19.0.0")
	assertProjectTechField(t, "evidence_path", tech.EvidencePath, "web/package.json")
	assertProjectTechField(t, "evidence_title", tech.EvidenceTitle, "Package manifest")

	firstObservation := projects[0].Observations[0]
	assertProjectTechField(t, "observation name", firstObservation.Name, "React")
	assertProjectTechField(t, "observation source", firstObservation.Source, "adr")
	assertProjectTechField(t, "observation evidence_path", firstObservation.EvidencePath, "docs/adrs/0007-adopt-react.md")
	assertProjectTechField(t, "observation evidence_title", firstObservation.EvidenceTitle, "ADR-0007 Adopt React")
	assertProjectTechField(t, "observation kind", firstObservation.ObservationKind, "adr")
	if firstObservation.ObservedAt == nil || !firstObservation.ObservedAt.Equal(now) {
		t.Fatalf("expected first observation observed_at %v, got %v", now, firstObservation.ObservedAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func assertProjectTechField(t *testing.T, field string, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("expected %s %q, got %q", field, want, got)
	}
}
