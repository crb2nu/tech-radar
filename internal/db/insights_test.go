package db

import (
	"testing"
	"time"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestBuildInsights(t *testing.T) {
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	radar := &types.Radar{
		Entries: []types.Entry{
			{
				Name:        "React",
				Quadrant:    types.QuadrantFrameworks,
				Ring:        types.RingAdopt,
				ProjectIDs:  []string{"alpha", "beta"},
				Tags:        []string{"frontend", "ui"},
				Description: "Primary UI framework",
				History: []types.HistoryEntry{
					{Date: now, Ring: types.RingAdopt, Reason: "Standardized", Author: "arch"},
					{Date: now.AddDate(0, -3, 0), Ring: types.RingTrial, Reason: "Piloted"},
				},
			},
			{
				Name:       "Terraform",
				Quadrant:   types.QuadrantTools,
				Ring:       types.RingHold,
				ProjectIDs: []string{"beta"},
				Tags:       []string{"infra"},
				History: []types.HistoryEntry{
					{Date: now.AddDate(0, 0, -10), Ring: types.RingHold, Reason: "Moving to GitOps", Author: "arch"},
					{Date: now.AddDate(0, -6, 0), Ring: types.RingTrial, Reason: "Infra baseline"},
				},
			},
		},
	}

	projects := []types.Project{
		{
			ID:       "1",
			Name:     "alpha",
			LastScan: now,
			Technologies: []types.DiscoveredTech{
				{Name: "React", Source: "package.json"},
				{Name: "TypeScript", Source: "package.json"},
			},
		},
		{
			ID:       "2",
			Name:     "beta",
			LastScan: now,
			Technologies: []types.DiscoveredTech{
				{Name: "React", Source: "package.json"},
				{Name: "Terraform", Source: "go.mod"},
			},
		},
	}

	insights := buildInsights(radar, projects)

	if insights.ProjectsCount != 2 {
		t.Fatalf("expected 2 projects, got %d", insights.ProjectsCount)
	}

	if insights.Summary.RadarCoveragePercent != 66.7 {
		t.Fatalf("expected coverage 66.7, got %.1f", insights.Summary.RadarCoveragePercent)
	}

	if len(insights.TopUncharted) != 1 || insights.TopUncharted[0].Name != "TypeScript" {
		t.Fatalf("expected TypeScript as uncharted tech, got %#v", insights.TopUncharted)
	}

	if len(insights.TopDrift) != 1 || insights.TopDrift[0].Name != "Terraform" {
		t.Fatalf("expected Terraform drift, got %#v", insights.TopDrift)
	}

	if len(insights.RecentMoves) == 0 || insights.RecentMoves[0].Name != "React" {
		t.Fatalf("expected React in recent moves, got %#v", insights.RecentMoves)
	}

	if len(insights.ProjectHealth) == 0 || insights.ProjectHealth[0].Name != "beta" {
		t.Fatalf("expected beta to be highest-risk project, got %#v", insights.ProjectHealth)
	}

	if insights.ProjectHealth[0].RiskLevel != "critical" {
		t.Fatalf("expected beta risk level to be critical, got %s", insights.ProjectHealth[0].RiskLevel)
	}
}
