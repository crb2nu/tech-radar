package radar

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.flexinfer.ai/services/tech-radar/internal/api"
	"gitlab.flexinfer.ai/services/tech-radar/internal/codebase"
	"gitlab.flexinfer.ai/services/tech-radar/internal/db"
	"gitlab.flexinfer.ai/services/tech-radar/internal/metrics"
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

	mux := s.routes(distFS)

	fmt.Printf("Starting radar server on http://0.0.0.0:%d\n", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux)
}

func (s *Server) routes(distFS fs.FS) *http.ServeMux {
	mux := http.NewServeMux()

	// Serve static files with proper cache control
	staticHandler := http.FileServer(http.FS(distFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Prevent aggressive caching of the index HTML
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else {
			// Asset files (which have cache-buster hashes) can be cached long term
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}

		if r.Method == http.MethodGet && path.Ext(r.URL.Path) == "" {
			serveIndex(w, r, distFS)
			return
		}

		staticHandler.ServeHTTP(w, r)
	})

	// Health endpoints. /livez always returns 200 if the process is alive
	// (kubelet won't restart-loop on transient DB outages). /healthz pings
	// the DB and is the right signal for readiness/traffic gating.
	mux.HandleFunc("/livez", s.handleLivez)
	mux.HandleFunc("/healthz", s.handleHealthz)

	// Prometheus scrape endpoint.
	mux.Handle("/metrics", promhttp.Handler())

	// API for radar data (Legacy compatibility)
	mux.HandleFunc("/api/radar", s.handleRadarData)

	// API v1 (Enterprise features)
	mux.HandleFunc("/api/v1/radar", s.handleV1Radar)
	mux.Handle("/api/v1/ingest", metrics.InstrumentIngest(http.HandlerFunc(s.apiServer.IngestScan)))
	mux.HandleFunc("/api/v1/entry", s.apiServer.UpsertEntry)
	mux.HandleFunc("/api/v1/search", s.apiServer.Search)
	mux.HandleFunc("/api/v1/projects", s.handleV1Projects)
	mux.HandleFunc("/api/v1/insights", s.handleV1Insights)
	mux.HandleFunc("/api/v1/codebase", s.handleV1Codebase)

	return mux
}

func (s *Server) handleLivez(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "mode": "file"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		log.Printf("healthz: db ping failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"reason": "database unreachable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "mode": "db"})
}

func (s *Server) handleRadarData(w http.ResponseWriter, r *http.Request) {
	radar, err := s.loadFileRadar()
	if err != nil {
		http.Error(w, err.Error(), statusForRadarLoad(err))
		return
	}

	writeJSON(w, http.StatusOK, radar)
}

func (s *Server) handleV1Radar(w http.ResponseWriter, r *http.Request) {
	if s.db != nil {
		s.apiServer.GetRadar(w, r)
		return
	}

	s.handleRadarData(w, r)
}

func (s *Server) handleV1Projects(w http.ResponseWriter, r *http.Request) {
	if s.db != nil {
		s.apiServer.GetProjects(w, r)
		return
	}

	writeJSON(w, http.StatusOK, []types.Project{})
}

func (s *Server) handleV1Insights(w http.ResponseWriter, r *http.Request) {
	if s.db != nil {
		s.apiServer.GetInsights(w, r)
		return
	}

	radar, err := s.loadFileRadar()
	if err != nil {
		http.Error(w, err.Error(), statusForRadarLoad(err))
		return
	}

	writeJSON(w, http.StatusOK, types.Insights{
		TechCount: len(radar.Entries),
	})
}

func (s *Server) handleV1Codebase(w http.ResponseWriter, r *http.Request) {
	if s.db != nil {
		s.apiServer.GetCodebaseAnalysis(w, r)
		return
	}

	radar, err := s.loadFileRadar()
	if err != nil {
		http.Error(w, err.Error(), statusForRadarLoad(err))
		return
	}

	writeJSON(w, http.StatusOK, codebase.Analyze(radar, nil))
}

func (s *Server) loadFileRadar() (*types.Radar, error) {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "radar.yaml"))
	if err != nil {
		return nil, fmt.Errorf("radar not initialized")
	}

	var radar types.Radar
	if err := yaml.Unmarshal(data, &radar); err != nil {
		return nil, fmt.Errorf("error parsing radar data")
	}

	return &radar, nil
}

func statusForRadarLoad(err error) int {
	if err.Error() == "radar not initialized" {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

func serveIndex(w http.ResponseWriter, r *http.Request, distFS fs.FS) {
	index, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	_, _ = w.Write(index)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
