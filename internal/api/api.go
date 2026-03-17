package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/meilisearch/meilisearch-go"
	"gitlab.flexinfer.ai/services/tech-radar/internal/db"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

// Server handles radar and ingestion API.
type Server struct {
	db    *db.DB
	meili meilisearch.ServiceManager
}

// NewServer creates a new API Server.
func NewServer(database *db.DB) *Server {
	var meiliClient meilisearch.ServiceManager
	if host := os.Getenv("MEILI_URL"); host != "" {
		meiliClient = meilisearch.New(host, meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))
	}

	return &Server{
		db:    database,
		meili: meiliClient,
	}
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

	// Process ingestion in background
	go s.processIngestion(result)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (s *Server) processIngestion(result types.ScanResult) {
	if s.db != nil {
		if err := s.db.SaveScan(result); err != nil {
			log.Printf("Error saving scan: %v", err)
			return
		}

		// Requirement 2: Drift Detection
		s.detectDrift(result)
	}

	// Requirement 1: Meilisearch Indexing
	if s.meili != nil {
		index := s.meili.Index("projects")
		index.AddDocuments(result, nil)
	}
}

func (s *Server) detectDrift(result types.ScanResult) {
	radar, err := s.db.GetRadar("default")
	if err != nil {
		return
	}

	// Map of technologies on HOLD
	onHold := make(map[string]bool)
	for _, e := range radar.Entries {
		if e.Ring == types.RingHold {
			onHold[e.Name] = true
		}
	}

	for _, tech := range result.Technologies {
		if onHold[tech.Name] {
			// Alert: Team using a technology on HOLD
			log.Printf("DRIFT DETECTED: Project %s is using %s which is on HOLD", result.ProjectID, tech.Name)
			// TODO: Send Slack/Webhook alert
		}
	}
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

	// Sync with Meilisearch
	if s.meili != nil {
		index := s.meili.Index("entries")
		index.AddDocuments(entry, nil)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(entry)
}

// Search handles full-text search across entries and projects.
func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' required", http.StatusBadRequest)
		return
	}

	if s.meili == nil {
		http.Error(w, "Search not configured", http.StatusInternalServerError)
		return
	}

	// Search entries
	entryRes, _ := s.meili.Index("entries").Search(query, &meilisearch.SearchRequest{
		Limit: 10,
	})

	// Search projects
	projectRes, _ := s.meili.Index("projects").Search(query, &meilisearch.SearchRequest{
		Limit: 10,
	})

	res := map[string]interface{}{
		"entries":  entryRes,
		"projects": projectRes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// GetProjects returns all projects and their discovered technologies.
func (s *Server) GetProjects(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	projects, err := s.db.GetProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
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
