package api

import (
	"encoding/json"
	"net/http"

	"gitlab.flexinfer.ai/services/tech-radar/internal/db"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

// Server handles radar and ingestion API.
type Server struct {
	db *db.DB
}

// NewServer creates a new API Server.
func NewServer(database *db.DB) *Server {
	return &Server{db: database}
}

// IngestScan handles technology scan results from CLI.
func (s *Server) IngestScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Validate API Key from Authorization header

	var result types.ScanResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process ingestion in background (as per design)
	go s.processIngestion(result)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (s *Server) processIngestion(result types.ScanResult) {
	if s.db != nil {
		s.db.SaveScan(result)
	}
	// TODO: Update Meilisearch
}

// UpsertEntry adds or updates a radar entry.
func (s *Server) UpsertEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var entry types.Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if s.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	if err := s.db.UpsertEntry(entry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entry)
}

// GetRadar returns the radar data.
func (s *Server) GetRadar(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = "default"
	}

	if s.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	radar, err := s.db.GetRadar(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(radar)
}
