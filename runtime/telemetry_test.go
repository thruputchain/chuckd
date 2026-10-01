package runtime

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/pprof"
	"reflect"
	"strings"
	"testing"
)

func TestTelemetryMuxDisabled(t *testing.T) {
	mux := telemetryMux(PprofConfig{})
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
	mux := telemetryMux(PprofConfig{Enabled: true})
	for _, name := range []string{"", "symbol", "heap", "allocs", "goroutine", "mutex", "block", "threadcreate"} {
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
		telemetryMux(PprofConfig{Enabled: enabled}).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if r.Code != http.StatusOK {
			t.Fatalf("enabled=%v: metrics status = %d", enabled, r.Code)
		}
	}
}

func TestTelemetryCmdlineNotFound(t *testing.T) {
	for _, path := range []string{"/debug/pprof/cmdline", "/debug/pprof/cmdline?debug=1", "/debug/pprof/cmdline/"} {
		r := httptest.NewRecorder()
		telemetryMux(PprofConfig{Enabled: true}).ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, r.Code)
		}
	}
}

func TestTelemetryProfileConflicts(t *testing.T) {
	for _, cpu := range []bool{false, true} {
		for _, trace := range []bool{false, true} {
			mux := telemetryMux(PprofConfig{Enabled: true, CPUProfileActive: cpu, TraceProfileActive: trace})
			for _, endpoint := range []struct {
				name   string
				flag   string
				active bool
			}{{"profile", "--cpu", cpu}, {"trace", "--trace", trace}} {
				req := httptest.NewRequest(http.MethodGet, "/debug/pprof/"+endpoint.name, nil)
				handler, _ := mux.Handler(req)
				if !endpoint.active {
					want := http.HandlerFunc(pprof.Profile)
					if endpoint.name == "trace" {
						want = pprof.Trace
					}
					if reflect.ValueOf(handler).Pointer() != reflect.ValueOf(want).Pointer() {
						t.Fatalf("%s: live handler unavailable", endpoint.name)
					}
					continue
				}
				r := httptest.NewRecorder()
				handler.ServeHTTP(r, req)
				if !strings.Contains(r.Body.String(), endpoint.flag) {
					t.Fatalf("%s: missing explanation: %s", endpoint.name, r.Body.String())
				}
				if r.Code != http.StatusConflict {
					t.Fatalf("cpu=%v trace=%v %s: status = %d, want 409", cpu, trace, endpoint.name, r.Code)
				}
			}
		}
	}
}
