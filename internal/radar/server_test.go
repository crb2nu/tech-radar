package radar

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"gitlab.flexinfer.ai/services/tech-radar/pkg/types"
)

func TestRoutesServeIndexForClientRoute(t *testing.T) {
	s := &Server{}
	mux := s.routes(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>radar app</html>")},
	})

	req := httptest.NewRequest(http.MethodGet, "/radar", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "<html>radar app</html>" {
		t.Fatalf("unexpected body %q", got)
	}
}

func TestRoutesServeFileBackedCodebaseWhenDatabaseMissing(t *testing.T) {
	dataDir := t.TempDir()
	writeRadarFile(t, dataDir)

	s := &Server{dataDir: dataDir}
	mux := s.routes(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>radar app</html>")},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/codebase", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var payload types.CodebaseAnalysis
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Summary.TotalTechnologies != 0 {
		t.Fatalf("expected no project-derived technologies, got %d", payload.Summary.TotalTechnologies)
	}
}

func writeRadarFile(t *testing.T, dataDir string) {
	t.Helper()
	data := []byte(`meta:
  name: Personal Tech Radar
entries:
  - id: go
    name: Go
    quadrant: languages
    ring: adopt
    moved: 0
`)
	if err := os.WriteFile(filepath.Join(dataDir, "radar.yaml"), data, fs.FileMode(0644)); err != nil {
		t.Fatal(err)
	}
}

func TestHandleLivez(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()

	s.handleLivez(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleHealthzNoDB(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	s.handleHealthz(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 in file mode, got %d", w.Code)
	}
}
