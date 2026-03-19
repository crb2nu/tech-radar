package db

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestSaveScanPersistsEvidenceFields(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	database := &DB{pool: mockDB}
	result := types.ScanResult{
		ProjectID: "alpha",
		Technologies: []types.DiscoveredTech{
			{
				Name:          "React",
				Version:       "^19.0.0",
				Source:        "adr",
				EvidencePath:  "docs/adrs/0007-adopt-react.md",
				EvidenceTitle: "ADR-0007 Adopt React",
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
	mock.ExpectExec(regexp.QuoteMeta(`
				INSERT INTO project_technologies (project_id, tech_name, version, source, evidence_path, evidence_title)
				VALUES ($1, $2, $3, $4, $5, $6)
			`)).
		WithArgs(
			"project-1",
			"React",
			"^19.0.0",
			"adr",
			"docs/adrs/0007-adopt-react.md",
			"ADR-0007 Adopt React",
		).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := database.SaveScan(result); err != nil {
		t.Fatalf("SaveScan returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestGetProjectsReturnsEvidenceFields(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	database := &DB{pool: mockDB}
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	projectRows := sqlmock.NewRows([]string{"id", "name", "url", "last_scan"}).
		AddRow("project-1", "alpha", "https://example.com/alpha", now)
	techRows := sqlmock.NewRows([]string{"tech_name", "version", "source", "evidence_path", "evidence_title"}).
		AddRow("React", "^19.0.0", "adr", "docs/adrs/0007-adopt-react.md", "ADR-0007 Adopt React")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, url, last_scan FROM projects ORDER BY name ASC")).
		WillReturnRows(projectRows)
	mock.ExpectQuery(regexp.QuoteMeta(`
				SELECT tech_name, version, source, evidence_path, evidence_title
				FROM project_technologies
				WHERE project_id = $1
			`)).
		WithArgs("project-1").
		WillReturnRows(techRows)

	projects, err := database.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects returned error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if len(projects[0].Technologies) != 1 {
		t.Fatalf("expected 1 technology, got %d", len(projects[0].Technologies))
	}

	tech := projects[0].Technologies[0]
	assertProjectTechField(t, "name", tech.Name, "React")
	assertProjectTechField(t, "source", tech.Source, "adr")
	assertProjectTechField(t, "version", tech.Version, "^19.0.0")
	assertProjectTechField(t, "evidence_path", tech.EvidencePath, "docs/adrs/0007-adopt-react.md")
	assertProjectTechField(t, "evidence_title", tech.EvidenceTitle, "ADR-0007 Adopt React")

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
