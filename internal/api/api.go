package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"gitlab.flexinfer.ai/services/tech-radar/internal/codebase"
	"gitlab.flexinfer.ai/services/tech-radar/internal/db"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

// Server handles radar and ingestion API.
type Server struct {
	db             *db.DB
	meili          meilisearch.ServiceManager
	hasAPIKeys     func() (bool, error)
	validateAPIKey func(string) (bool, error)
	driftWebhook   string
	webhookClient  *http.Client
}

// NewServer creates a new API Server.
func NewServer(database *db.DB) *Server {
	var meiliClient meilisearch.ServiceManager
	if host := os.Getenv("MEILI_URL"); host != "" {
		meiliClient = meilisearch.New(host, meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")))
	}

	var hasAPIKeys func() (bool, error)
	var validateAPIKey func(string) (bool, error)
	if database != nil {
		hasAPIKeys = database.HasAPIKeys
		validateAPIKey = database.ValidateAPIKey
	}

	return &Server{
		db:             database,
		meili:          meiliClient,
		hasAPIKeys:     hasAPIKeys,
		validateAPIKey: validateAPIKey,
		driftWebhook:   strings.TrimSpace(os.Getenv("DRIFT_WEBHOOK_URL")),
		webhookClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

// IngestScan handles technology scan results from CLI.
func (s *Server) IngestScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.authorizeIngestRequest(r); err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, errAPIKeyValidationUnavailable) {
			status = http.StatusInternalServerError
			log.Printf("ingest: auth check failed (returning 500): %v", err)
		}
		http.Error(w, err.Error(), status)
		return
	}

	var result types.ScanResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		log.Printf("ingest: invalid payload from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process ingestion in background
	go s.processIngestion(result)

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

var errAPIKeyValidationUnavailable = errors.New("api key validation unavailable")

func (s *Server) authorizeIngestRequest(r *http.Request) error {
	if s.hasAPIKeys == nil {
		return nil
	}

	required, err := s.hasAPIKeys()
	if err != nil {
		return fmt.Errorf("%w: hasAPIKeys: %v", errAPIKeyValidationUnavailable, err)
	}
	if !required {
		return nil
	}

	apiKey := extractAPIKey(r)
	if apiKey == "" {
		return errors.New("missing API key")
	}
	if s.validateAPIKey == nil {
		return errAPIKeyValidationUnavailable
	}

	valid, err := s.validateAPIKey(apiKey)
	if err != nil {
		return fmt.Errorf("%w: validateAPIKey: %v", errAPIKeyValidationUnavailable, err)
	}
	if !valid {
		return errors.New("invalid API key")
	}

	return nil
}

func extractAPIKey(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[len("Bearer "):])
	}

	return strings.TrimSpace(r.Header.Get("X-API-Key"))
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
		if _, err := index.AddDocuments(result, nil); err != nil {
			log.Printf("Error indexing project scan in Meilisearch: %v", err)
		}
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
			log.Printf("DRIFT DETECTED: Project %s is using %s which is on HOLD", result.ProjectID, tech.Name)
			s.sendDriftWebhook(result.ProjectID, tech.Name)
		}
	}
}

// driftEvent is the JSON payload posted to DRIFT_WEBHOOK_URL. The "text"
// field is filled in for compatibility with Slack-style incoming webhooks;
// other consumers can ignore it and read the structured fields.
type driftEvent struct {
	Type       string    `json:"type"`
	Text       string    `json:"text"`
	ProjectID  string    `json:"project_id"`
	Technology string    `json:"technology"`
	Ring       string    `json:"ring"`
	DetectedAt time.Time `json:"detected_at"`
}

func (s *Server) sendDriftWebhook(projectID, tech string) {
	if s.driftWebhook == "" {
		return
	}

	event := driftEvent{
		Type:       "drift",
		Text:       fmt.Sprintf("DRIFT: project %s is using %s which is on HOLD", projectID, tech),
		ProjectID:  projectID,
		Technology: tech,
		Ring:       string(types.RingHold),
		DetectedAt: time.Now().UTC(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("drift webhook: marshal failed: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, s.driftWebhook, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("drift webhook: build request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.webhookClient.Do(req)
	if err != nil {
		log.Printf("drift webhook: post failed: %v", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		log.Printf("drift webhook: %s returned status %s", s.driftWebhook, resp.Status)
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
		if _, err := index.AddDocuments(entry, nil); err != nil {
			log.Printf("Error indexing entry in Meilisearch: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, entry)
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

	writeJSON(w, http.StatusOK, res)
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

	writeJSON(w, http.StatusOK, projects)
}

// GetInsights returns ecosystem-wide technology adoption analytics.
func (s *Server) GetInsights(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	insights, err := s.db.GetInsights()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, insights)
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

	writeJSON(w, http.StatusOK, radar)
}

// GetCodebaseAnalysis returns the codebase visualization payload.
func (s *Server) GetCodebaseAnalysis(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	radar, err := s.db.GetRadar("default")
	if err != nil {
		radar = &types.Radar{}
	}

	projects, err := s.db.GetProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	analysis := codebase.Analyze(radar, projects)
	writeJSON(w, http.StatusOK, analysis)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
