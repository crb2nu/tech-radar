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
