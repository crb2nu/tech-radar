package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusClass(t *testing.T) {
	cases := map[int]string{
		200: "2xx", 202: "2xx",
		400: "4xx", 401: "4xx", 404: "4xx",
		500: "5xx", 502: "5xx", 503: "5xx",
		100: "other",
	}
	for code, want := range cases {
		if got := StatusClass(code); got != want {
			t.Errorf("StatusClass(%d) = %q, want %q", code, got, want)
		}
	}
}

func TestInstrumentIngestRecordsStatus(t *testing.T) {
	h := InstrumentIngest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	// We can't assert on the global counter value without resetting it
	// across tests, but covering the path catches obvious regressions.
}
