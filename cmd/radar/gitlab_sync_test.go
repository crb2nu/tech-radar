package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListGitLabMembershipProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "secret-token" {
			t.Fatalf("expected GitLab token header, got %q", got)
		}
		if got := r.URL.Query().Get("membership"); got != "true" {
			t.Fatalf("expected membership=true, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]gitLabProject{{
			ID:                1,
			PathWithNamespace: "services/tech-radar",
			HTTPURLToRepo:     "https://gitlab.example.com/services/tech-radar.git",
			DefaultBranch:     "main",
		}}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	projects, err := listGitLabMembershipProjects(server.URL, "secret-token", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].PathWithNamespace != "services/tech-radar" {
		t.Fatalf("unexpected project %+v", projects[0])
	}
}

func TestAuthenticatedCloneURL(t *testing.T) {
	cloneURL, err := authenticatedCloneURL("https://gitlab.example.com/services/tech-radar.git", "secret-token")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(cloneURL, "https://oauth2:secret-token@gitlab.example.com/") {
		t.Fatalf("unexpected authenticated clone URL %q", cloneURL)
	}
}
