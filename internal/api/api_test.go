package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestIngestScan(t *testing.T) {
	s := NewServer(nil)

	result := types.ScanResult{
		ProjectID: "test-project",
		Technologies: []types.DiscoveredTech{
			{Name: "Go", Source: "go.mod", Version: "1.24"},
		},
	}

	data, _ := json.Marshal(result)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(data))
	w := httptest.NewRecorder()

	s.IngestScan(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}
}

func TestIngestScanRequiresAPIKeyWhenConfigured(t *testing.T) {
	s := NewServer(nil)
	s.hasAPIKeys = func() (bool, error) { return true, nil }
	s.validateAPIKey = func(apiKey string) (bool, error) { return apiKey == "secret-token", nil }

	result := types.ScanResult{
		ProjectID: "test-project",
		Technologies: []types.DiscoveredTech{
			{Name: "Go", Source: "go.mod", Version: "1.24"},
		},
	}

	data, _ := json.Marshal(result)

	t.Run("rejects missing key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(data))
		w := httptest.NewRecorder()

		s.IngestScan(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("accepts bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer secret-token")
		w := httptest.NewRecorder()

		s.IngestScan(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d", w.Code)
		}
	})

	t.Run("accepts x-api-key for compatibility", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(data))
		req.Header.Set("X-API-Key", "secret-token")
		w := httptest.NewRecorder()

		s.IngestScan(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("expected status 202, got %d", w.Code)
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer wrong-token")
		w := httptest.NewRecorder()

		s.IngestScan(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestSendDriftWebhookNoOpWhenUnconfigured(t *testing.T) {
	s := NewServer(nil)
	s.driftWebhook = ""
	// Should not panic or attempt any network call.
	s.sendDriftWebhook("services/foo", "PHP")
}

func TestSendDriftWebhookPostsPayload(t *testing.T) {
	var got driftEvent
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewServer(nil)
	s.driftWebhook = srv.URL

	s.sendDriftWebhook("services/foo", "PHP")

	if got.Type != "drift" {
		t.Errorf("type=%q, want drift", got.Type)
	}
	if got.ProjectID != "services/foo" || got.Technology != "PHP" {
		t.Errorf("got %+v", got)
	}
	if got.Text == "" {
		t.Error("expected human-readable text field")
	}
}
