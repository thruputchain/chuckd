package runtime

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelemetryMuxDisabled(t *testing.T) {
	mux := telemetryMux(false)
	for _, name := range []string{"", "cmdline", "profile", "symbol", "trace", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/debug/pprof/"+name, nil))
			if r.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", r.Code)
			}
		})
	}
}

func TestTelemetryMuxEnabled(t *testing.T) {
	mux := telemetryMux(true)
	for _, name := range []string{"", "cmdline", "symbol", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/debug/pprof/"+name, nil))
		if r.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", name, r.Code)
		}
		if name == "heap" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(gz)
			gz.Close()
			if err != nil || len(body) == 0 {
				t.Fatalf("invalid heap profile: length=%d, error=%v", len(body), err)
			}
		}
	}
	for _, name := range []string{"profile", "trace"} {
		req := httptest.NewRequest(http.MethodGet, "/debug/pprof/"+name, nil)
		_, pattern := mux.Handler(req)
		if pattern != "/debug/pprof/"+name {
			t.Fatalf("%s: registered pattern = %q", name, pattern)
		}
	}
}

func TestTelemetryMetrics(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		r := httptest.NewRecorder()
		telemetryMux(enabled).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("enabled=%v: metrics status = %d", enabled, r.Code)
		}
	}
}
