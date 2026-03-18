package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestPublishResultSetsAuthHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "secret-token" {
			t.Fatalf("expected X-API-Key header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	}))
	defer server.Close()

	err := publishResult(server.URL, "secret-token", types.ScanResult{
		ProjectID: "demo",
		Technologies: []types.DiscoveredTech{
			{Name: "Go", Source: "go.mod"},
		},
	})
	if err != nil {
		t.Fatalf("publishResult returned error: %v", err)
	}
}
