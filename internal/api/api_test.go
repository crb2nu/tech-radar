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
