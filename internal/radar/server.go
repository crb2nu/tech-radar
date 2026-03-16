package radar

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"gitlab.flexinfer.ai/services/tech-radar/internal/api"
	"gitlab.flexinfer.ai/services/tech-radar/internal/db"
	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
	"gopkg.in/yaml.v3"
)

// Server handles serving the radar visualization and data.
type Server struct {
	port      int
	dataDir   string
	assets    embed.FS
	db        *db.DB
	apiServer *api.Server
}

// NewServer creates a new Server.
func NewServer(port int, dataDir string, assets embed.FS) *Server {
	// Optional database from env
	var database *db.DB
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		if d, err := db.New(dsn); err == nil {
			database = d
		} else {
			fmt.Printf("Warning: Failed to connect to database: %v\n", err)
		}
	}

	return &Server{
		port:      port,
		dataDir:   dataDir,
		assets:    assets,
		db:        database,
		apiServer: api.NewServer(database),
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	// The path in Sub must match the path in go:embed
	distFS, err := fs.Sub(s.assets, "web/dist")
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	// Serve static files
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// API for radar data (Legacy compatibility)
	mux.HandleFunc("/api/radar", s.handleRadarData)

	// API v1 (Enterprise features)
	mux.HandleFunc("/api/v1/radar", s.apiServer.GetRadar)
	mux.HandleFunc("/api/v1/ingest", s.apiServer.IngestScan)
	mux.HandleFunc("/api/v1/entry", s.apiServer.UpsertEntry)
	mux.HandleFunc("/api/v1/search", s.apiServer.Search)

	fmt.Printf("Starting radar server on http://0.0.0.0:%d\n", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux)
}

func (s *Server) handleRadarData(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "radar.yaml"))
	if err != nil {
		http.Error(w, "Radar not initialized", http.StatusNotFound)
		return
	}

	var radar types.Radar
	if err := yaml.Unmarshal(data, &radar); err != nil {
		http.Error(w, "Error parsing radar data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(radar)
}
